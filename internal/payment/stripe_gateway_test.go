package payment

import (
	"errors"
	"testing"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
)

const testWebhookSecret = "whsec_test"

func signedEvent(t *testing.T, payload string, secret string, ts time.Time) ([]byte, string) {
	t.Helper()
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   []byte(payload),
		Secret:    secret,
		Timestamp: ts,
	})
	return signed.Payload, signed.Header
}

func TestParseWebhookEvent_PaymentIntentEvent(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret)
	payload, header := signedEvent(t, `{
		"id": "evt_1", "object": "event", "type": "payment_intent.succeeded", "api_version": "2020-08-27",
		"data": {"object": {"id": "pi_123", "object": "payment_intent", "status": "succeeded"}}
	}`, testWebhookSecret, time.Now())

	event, err := g.ParseWebhookEvent(payload, header)

	assert.NoError(t, err)
	assert.Equal(t, domain.PaymentEventSucceeded, event.Type)
	assert.Equal(t, "pi_123", event.PaymentIntentID)
}

func TestParseWebhookEvent_OtherEventHasNoIntent(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret)
	payload, header := signedEvent(t, `{
		"id": "evt_2", "object": "event", "type": "customer.created",
		"data": {"object": {"id": "cus_1", "object": "customer"}}
	}`, testWebhookSecret, time.Now())

	event, err := g.ParseWebhookEvent(payload, header)

	assert.NoError(t, err)
	assert.Equal(t, domain.PaymentEventType("customer.created"), event.Type)
	assert.Empty(t, event.PaymentIntentID)
}

func TestParseWebhookEvent_WrongSecret(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret)
	payload, header := signedEvent(t, `{"id": "evt_3", "object": "event", "type": "payment_intent.succeeded"}`, "whsec_other", time.Now())

	_, err := g.ParseWebhookEvent(payload, header)

	assert.ErrorIs(t, err, domain.ErrInvalidWebhookSignature)
}

func TestParseWebhookEvent_TamperedPayload(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret)
	_, header := signedEvent(t, `{"id": "evt_4", "object": "event", "type": "payment_intent.canceled"}`, testWebhookSecret, time.Now())

	_, err := g.ParseWebhookEvent([]byte(`{"id": "evt_4", "object": "event", "type": "payment_intent.succeeded"}`), header)

	assert.ErrorIs(t, err, domain.ErrInvalidWebhookSignature)
}

func TestParseWebhookEvent_ReplayedOldEvent(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret)
	payload, header := signedEvent(t, `{"id": "evt_5", "object": "event", "type": "payment_intent.succeeded"}`, testWebhookSecret, time.Now().Add(-time.Hour))

	_, err := g.ParseWebhookEvent(payload, header)

	assert.ErrorIs(t, err, domain.ErrInvalidWebhookSignature)
}

func TestWrapStripeError(t *testing.T) {
	unexpected := wrapStripeError(&stripe.Error{Code: stripe.ErrorCodePaymentIntentUnexpectedState, Msg: "already succeeded"})
	assert.ErrorIs(t, unexpected, domain.ErrPaymentIntentUnexpectedState)
	assert.NotErrorIs(t, unexpected, domain.ErrPaymentProvider)

	other := wrapStripeError(errors.New("connection refused"))
	assert.ErrorIs(t, other, domain.ErrPaymentProvider)
}
