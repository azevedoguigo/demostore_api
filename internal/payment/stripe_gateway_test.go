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
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
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
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
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
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
	payload, header := signedEvent(t, `{"id": "evt_3", "object": "event", "type": "payment_intent.succeeded"}`, "whsec_other", time.Now())

	_, err := g.ParseWebhookEvent(payload, header)

	assert.ErrorIs(t, err, domain.ErrInvalidWebhookSignature)
}

func TestParseWebhookEvent_TamperedPayload(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
	_, header := signedEvent(t, `{"id": "evt_4", "object": "event", "type": "payment_intent.canceled"}`, testWebhookSecret, time.Now())

	_, err := g.ParseWebhookEvent([]byte(`{"id": "evt_4", "object": "event", "type": "payment_intent.succeeded"}`), header)

	assert.ErrorIs(t, err, domain.ErrInvalidWebhookSignature)
}

func TestParseWebhookEvent_ReplayedOldEvent(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
	payload, header := signedEvent(t, `{"id": "evt_5", "object": "event", "type": "payment_intent.succeeded"}`, testWebhookSecret, time.Now().Add(-time.Hour))

	_, err := g.ParseWebhookEvent(payload, header)

	assert.ErrorIs(t, err, domain.ErrInvalidWebhookSignature)
}

func TestParseWebhookEvent_BoletoIssued(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
	payload, header := signedEvent(t, `{
		"id": "evt_6", "object": "event", "type": "payment_intent.requires_action",
		"data": {"object": {"id": "pi_123", "object": "payment_intent", "status": "requires_action",
			"next_action": {"type": "boleto_display_details", "boleto_display_details": {"expires_at": 1767236399}}}}
	}`, testWebhookSecret, time.Now())

	event, err := g.ParseWebhookEvent(payload, header)

	assert.NoError(t, err)
	assert.Equal(t, domain.PaymentEventRequiresAction, event.Type)
	assert.Equal(t, "pi_123", event.PaymentIntentID)
	assert.Equal(t, domain.NextActionBoletoDisplayDetails, event.NextActionType)
	assert.Equal(t, int64(1767236399), event.NextActionExpiresAt.Unix())
}

func TestParseWebhookEvent_Refund(t *testing.T) {
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
	payload, header := signedEvent(t, `{
		"id": "evt_7", "object": "event", "type": "refund.updated",
		"data": {"object": {"id": "re_1", "object": "refund", "status": "succeeded", "metadata": {"refund_id": "abc"}}}
	}`, testWebhookSecret, time.Now())

	event, err := g.ParseWebhookEvent(payload, header)

	assert.NoError(t, err)
	assert.Equal(t, domain.PaymentEventRefundUpdated, event.Type)
	assert.Empty(t, event.PaymentIntentID)
	assert.Equal(t, &domain.ProviderRefund{ID: "re_1", Status: "succeeded", LocalRefundID: "abc"}, event.Refund)
}

func TestPixExpiresAt_AlignedWithOrderAndClamped(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	g := NewStripeGateway("sk_test", testWebhookSecret, 3)
	g.now = func() time.Time { return now }

	assert.Equal(t, now.Add(30*time.Minute).Unix(), g.pixExpiresAt(now.Add(30*time.Minute)))
	assert.Equal(t, now.Add(time.Minute).Unix(), g.pixExpiresAt(now.Add(5*time.Second)), "Stripe requires at least 10s ahead")
	assert.Equal(t, now.Add(72*time.Hour).Unix(), g.pixExpiresAt(now.Add(10*24*time.Hour)), "Stripe allows at most 3 days")
}

func TestWrapStripeError(t *testing.T) {
	unexpected := wrapStripeError(&stripe.Error{Code: stripe.ErrorCodePaymentIntentUnexpectedState, Msg: "already succeeded"})
	assert.ErrorIs(t, unexpected, domain.ErrPaymentIntentUnexpectedState)
	assert.NotErrorIs(t, unexpected, domain.ErrPaymentProvider)

	rejected := wrapStripeError(&stripe.Error{HTTPStatusCode: 400, Msg: "amount exceeds charge"})
	assert.ErrorIs(t, rejected, domain.ErrPaymentProvider)
	assert.ErrorIs(t, rejected, domain.ErrPaymentProviderRejected)

	for _, status := range []int{429, 500} {
		unknown := wrapStripeError(&stripe.Error{HTTPStatusCode: status})
		assert.ErrorIs(t, unknown, domain.ErrPaymentProvider)
		assert.NotErrorIs(t, unknown, domain.ErrPaymentProviderRejected, status)
	}

	network := wrapStripeError(errors.New("connection refused"))
	assert.ErrorIs(t, network, domain.ErrPaymentProvider)
	assert.NotErrorIs(t, network, domain.ErrPaymentProviderRejected)
}
