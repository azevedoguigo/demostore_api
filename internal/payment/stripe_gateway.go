package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
)

type StripeGateway struct {
	client        *stripe.Client
	webhookSecret string
}

func NewStripeGateway(secretKey, webhookSecret string) *StripeGateway {
	return &StripeGateway{
		client:        stripe.NewClient(secretKey),
		webhookSecret: webhookSecret,
	}
}

func toPaymentIntent(pi *stripe.PaymentIntent) *domain.PaymentIntent {
	return &domain.PaymentIntent{ID: pi.ID, ClientSecret: pi.ClientSecret, Status: string(pi.Status)}
}

func wrapStripeError(err error) error {
	var stripeErr *stripe.Error
	if errors.As(err, &stripeErr) && stripeErr.Code == stripe.ErrorCodePaymentIntentUnexpectedState {
		return fmt.Errorf("%w: %s", domain.ErrPaymentIntentUnexpectedState, stripeErr.Msg)
	}

	return fmt.Errorf("%w: %v", domain.ErrPaymentProvider, err)
}

// CreatePaymentIntent uses the order ID as idempotency key, so retries never create a second intent for the same order.
func (g *StripeGateway) CreatePaymentIntent(orderID uuid.UUID, amount int64, currency string) (*domain.PaymentIntent, error) {
	params := &stripe.PaymentIntentCreateParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentCreateAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	params.AddMetadata("order_id", orderID.String())
	params.SetIdempotencyKey("payment-intent-" + orderID.String())

	pi, err := g.client.V1PaymentIntents.Create(context.Background(), params)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return toPaymentIntent(pi), nil
}

func (g *StripeGateway) GetPaymentIntent(id string) (*domain.PaymentIntent, error) {
	pi, err := g.client.V1PaymentIntents.Retrieve(context.Background(), id, nil)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return toPaymentIntent(pi), nil
}

func (g *StripeGateway) CancelPaymentIntent(id string) error {
	if _, err := g.client.V1PaymentIntents.Cancel(context.Background(), id, nil); err != nil {
		return wrapStripeError(err)
	}

	return nil
}

// RefundPaymentIntent refunds the full amount. The idempotency key makes webhook retries safe.
func (g *StripeGateway) RefundPaymentIntent(id string) error {
	params := &stripe.RefundCreateParams{PaymentIntent: stripe.String(id)}
	params.SetIdempotencyKey("refund-" + id)

	if _, err := g.client.V1Refunds.Create(context.Background(), params); err != nil {
		return wrapStripeError(err)
	}

	return nil
}

func (g *StripeGateway) ParseWebhookEvent(payload []byte, signature string) (*domain.PaymentEvent, error) {
	// Only the event type and the payment intent ID are read, which are stable across API
	// versions, so a webhook endpoint configured with a different version is accepted.
	event, err := webhook.ConstructEventWithOptions(payload, signature, g.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidWebhookSignature, err)
	}

	result := &domain.PaymentEvent{Type: domain.PaymentEventType(event.Type)}
	if !strings.HasPrefix(string(event.Type), "payment_intent.") {
		return result, nil
	}

	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return nil, fmt.Errorf("failed to decode payment intent from event %s: %w", event.ID, err)
	}

	result.PaymentIntentID = pi.ID
	return result, nil
}
