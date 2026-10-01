package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
)

// Stripe accepts a Pix expiration between 10 seconds and 3 days in the future.
const (
	pixMinExpiration = time.Minute
	pixMaxExpiration = 72 * time.Hour
)

type StripeGateway struct {
	client                 *stripe.Client
	webhookSecret          string
	boletoExpiresAfterDays int64
	now                    func() time.Time
}

func NewStripeGateway(secretKey, webhookSecret string, boletoExpiresAfterDays int) *StripeGateway {
	return &StripeGateway{
		client:                 stripe.NewClient(secretKey),
		webhookSecret:          webhookSecret,
		boletoExpiresAfterDays: int64(boletoExpiresAfterDays),
		now:                    time.Now,
	}
}

func toPaymentIntent(pi *stripe.PaymentIntent) *domain.PaymentIntent {
	return &domain.PaymentIntent{ID: pi.ID, ClientSecret: pi.ClientSecret, Status: string(pi.Status)}
}

func wrapStripeError(err error) error {
	var stripeErr *stripe.Error
	if !errors.As(err, &stripeErr) {
		return fmt.Errorf("%w: %v", domain.ErrPaymentProvider, err)
	}

	if stripeErr.Code == stripe.ErrorCodePaymentIntentUnexpectedState {
		return fmt.Errorf("%w: %s", domain.ErrPaymentIntentUnexpectedState, stripeErr.Msg)
	}

	// 4xx responses (other than rate limiting) mean Stripe refused the request: retrying won't help
	// and nothing was created. Other failures leave the outcome unknown.
	if stripeErr.HTTPStatusCode >= 400 && stripeErr.HTTPStatusCode < 500 && stripeErr.HTTPStatusCode != http.StatusTooManyRequests {
		return fmt.Errorf("%w: %w: %s", domain.ErrPaymentProvider, domain.ErrPaymentProviderRejected, stripeErr.Msg)
	}

	return fmt.Errorf("%w: %v", domain.ErrPaymentProvider, err)
}

func (g *StripeGateway) pixExpiresAt(orderExpiresAt time.Time) int64 {
	now := g.now()
	expiresAt := orderExpiresAt
	if expiresAt.Before(now.Add(pixMinExpiration)) {
		expiresAt = now.Add(pixMinExpiration)
	}
	if expiresAt.After(now.Add(pixMaxExpiration)) {
		expiresAt = now.Add(pixMaxExpiration)
	}

	return expiresAt.Unix()
}

// CreatePaymentIntent uses the order ID as idempotency key, so retries never create a second intent for the same order.
func (g *StripeGateway) CreatePaymentIntent(req domain.PaymentIntentRequest) (*domain.PaymentIntent, error) {
	params := &stripe.PaymentIntentCreateParams{
		Amount:                    stripe.Int64(req.Amount),
		Currency:                  stripe.String(req.Currency),
		AllowedPaymentMethodTypes: stripe.StringSlice(req.PaymentMethodTypes),
	}

	options := &stripe.PaymentIntentCreatePaymentMethodOptionsParams{}
	for _, method := range req.PaymentMethodTypes {
		switch method {
		case domain.PaymentMethodPix:
			if req.ExpiresAt.IsZero() {
				continue // Stripe's default Pix expiration applies.
			}
			options.Pix = &stripe.PaymentIntentCreatePaymentMethodOptionsPixParams{
				ExpiresAt: stripe.Int64(g.pixExpiresAt(req.ExpiresAt)),
			}
		case domain.PaymentMethodBoleto:
			options.Boleto = &stripe.PaymentIntentCreatePaymentMethodOptionsBoletoParams{
				ExpiresAfterDays: stripe.Int64(g.boletoExpiresAfterDays),
			}
		}
	}
	params.PaymentMethodOptions = options

	params.AddMetadata("order_id", req.OrderID.String())
	params.SetIdempotencyKey("payment-intent-" + req.OrderID.String())

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

func (g *StripeGateway) GetPaymentMethodType(paymentIntentID string) (string, error) {
	params := &stripe.PaymentIntentRetrieveParams{}
	params.AddExpand("latest_charge")

	pi, err := g.client.V1PaymentIntents.Retrieve(context.Background(), paymentIntentID, params)
	if err != nil {
		return "", wrapStripeError(err)
	}

	if pi.LatestCharge == nil || pi.LatestCharge.PaymentMethodDetails == nil {
		return "", fmt.Errorf("%w: payment intent %s has no charge", domain.ErrPaymentProvider, paymentIntentID)
	}

	return string(pi.LatestCharge.PaymentMethodDetails.Type), nil
}

// RefundPaymentIntent uses our refund ID as idempotency key, so retrying a refund never refunds twice.
func (g *StripeGateway) RefundPaymentIntent(paymentIntentID string, amount int64, refundID uuid.UUID) (*domain.ProviderRefund, error) {
	params := &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(paymentIntentID),
		Amount:        stripe.Int64(amount),
	}
	params.AddMetadata("refund_id", refundID.String())
	params.SetIdempotencyKey("refund-" + refundID.String())

	refund, err := g.client.V1Refunds.Create(context.Background(), params)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	return &domain.ProviderRefund{ID: refund.ID, Status: string(refund.Status), LocalRefundID: refund.Metadata["refund_id"]}, nil
}

func (g *StripeGateway) ParseWebhookEvent(payload []byte, signature string) (*domain.PaymentEvent, error) {
	// Only the fields read below are used, which are stable across API versions, so a webhook
	// endpoint configured with a different version is accepted.
	event, err := webhook.ConstructEventWithOptions(payload, signature, g.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidWebhookSignature, err)
	}

	result := &domain.PaymentEvent{Type: domain.PaymentEventType(event.Type)}

	switch {
	case strings.HasPrefix(string(event.Type), "payment_intent."):
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("failed to decode payment intent from event %s: %w", event.ID, err)
		}

		result.PaymentIntentID = pi.ID
		if pi.NextAction != nil {
			result.NextActionType = string(pi.NextAction.Type)
			if details := pi.NextAction.BoletoDisplayDetails; details != nil && details.ExpiresAt > 0 {
				result.NextActionExpiresAt = time.Unix(details.ExpiresAt, 0)
			}
		}

	case strings.HasPrefix(string(event.Type), "refund."):
		var refund stripe.Refund
		if err := json.Unmarshal(event.Data.Raw, &refund); err != nil {
			return nil, fmt.Errorf("failed to decode refund from event %s: %w", event.ID, err)
		}

		result.Refund = &domain.ProviderRefund{ID: refund.ID, Status: string(refund.Status), LocalRefundID: refund.Metadata["refund_id"]}
	}

	return result, nil
}
