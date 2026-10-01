package payment

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"
)

type capturedRequest struct {
	method         string
	path           string
	form           url.Values
	idempotencyKey string
}

// fakeStripe serves canned JSON responses and records each request, so the gateway's use of the
// SDK (endpoints, form parameters, headers) is checked without calling Stripe.
func fakeStripe(t *testing.T, status int, body string) (*StripeGateway, *[]capturedRequest) {
	t.Helper()
	var requests []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		for key, values := range r.URL.Query() {
			form[key] = values
		}
		requests = append(requests, capturedRequest{r.Method, r.URL.Path, form, r.Header.Get("Idempotency-Key")})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	backends := stripe.NewBackendsWithConfig(&stripe.BackendConfig{
		URL:               stripe.String(server.URL),
		MaxNetworkRetries: stripe.Int64(0),
		LeveledLogger:     &stripe.LeveledLogger{Level: stripe.LevelNull},
	})

	g := NewStripeGateway("sk_test", testWebhookSecret, 5)
	g.client = stripe.NewClient("sk_test", stripe.WithBackends(backends))
	return g, &requests
}

func TestCreatePaymentIntent_SendsMethodsExpirationsAndIdempotencyKey(t *testing.T) {
	g, requests := fakeStripe(t, 200, `{"id": "pi_1", "object": "payment_intent", "client_secret": "pi_1_secret", "status": "requires_payment_method"}`)
	now := time.Unix(1_700_000_000, 0)
	g.now = func() time.Time { return now }
	orderID := uuid.New()

	pi, err := g.CreatePaymentIntent(domain.PaymentIntentRequest{
		OrderID:            orderID,
		Amount:             4028,
		Currency:           "brl",
		PaymentMethodTypes: []string{"card", "pix", "boleto"},
		ExpiresAt:          now.Add(30 * time.Minute),
	})

	require.NoError(t, err)
	assert.Equal(t, &domain.PaymentIntent{ID: "pi_1", ClientSecret: "pi_1_secret", Status: "requires_payment_method"}, pi)

	require.Len(t, *requests, 1)
	req := (*requests)[0]
	assert.Equal(t, "POST", req.method)
	assert.Equal(t, "/v1/payment_intents", req.path)
	assert.Equal(t, "payment-intent-"+orderID.String(), req.idempotencyKey)
	assert.Equal(t, "4028", req.form.Get("amount"))
	assert.Equal(t, "brl", req.form.Get("currency"))
	for i, method := range []string{"card", "pix", "boleto"} {
		assert.Equal(t, method, req.form.Get(fmt.Sprintf("allowed_payment_method_types[%d]", i)))
	}
	assert.Equal(t, fmt.Sprint(now.Add(30*time.Minute).Unix()), req.form.Get("payment_method_options[pix][expires_at]"))
	assert.Equal(t, "5", req.form.Get("payment_method_options[boleto][expires_after_days]"))
	assert.Equal(t, orderID.String(), req.form.Get("metadata[order_id]"))
	assert.Empty(t, req.form.Get("automatic_payment_methods[enabled]"))
}

func TestCreatePaymentIntent_CardOnlyHasNoMethodOptions(t *testing.T) {
	g, requests := fakeStripe(t, 200, `{"id": "pi_1", "object": "payment_intent"}`)

	_, err := g.CreatePaymentIntent(domain.PaymentIntentRequest{
		OrderID: uuid.New(), Amount: 10_000_000, Currency: "brl", PaymentMethodTypes: []string{"card"},
	})

	require.NoError(t, err)
	req := (*requests)[0]
	assert.Equal(t, "card", req.form.Get("allowed_payment_method_types[0]"))
	assert.Empty(t, req.form.Get("payment_method_options[pix][expires_at]"))
	assert.Empty(t, req.form.Get("payment_method_options[boleto][expires_after_days]"))
}

func TestCreatePaymentIntent_PixWithoutOrderExpirationUsesStripeDefault(t *testing.T) {
	g, requests := fakeStripe(t, 200, `{"id": "pi_1", "object": "payment_intent"}`)

	_, err := g.CreatePaymentIntent(domain.PaymentIntentRequest{
		OrderID: uuid.New(), Amount: 1000, Currency: "brl", PaymentMethodTypes: []string{"card", "pix"},
	})

	require.NoError(t, err)
	assert.Empty(t, (*requests)[0].form.Get("payment_method_options[pix][expires_at]"))
}

func TestRefundPaymentIntent_SendsAmountMetadataAndIdempotencyKey(t *testing.T) {
	g, requests := fakeStripe(t, 200, `{"id": "re_1", "object": "refund", "status": "pending", "metadata": {"refund_id": "x"}}`)
	refundID := uuid.New()

	refund, err := g.RefundPaymentIntent("pi_1", 1999, refundID)

	require.NoError(t, err)
	assert.Equal(t, &domain.ProviderRefund{ID: "re_1", Status: "pending", LocalRefundID: "x"}, refund)
	req := (*requests)[0]
	assert.Equal(t, "/v1/refunds", req.path)
	assert.Equal(t, "refund-"+refundID.String(), req.idempotencyKey)
	assert.Equal(t, "pi_1", req.form.Get("payment_intent"))
	assert.Equal(t, "1999", req.form.Get("amount"))
	assert.Equal(t, refundID.String(), req.form.Get("metadata[refund_id]"))
}

func TestRefundPaymentIntent_RejectionIsDefinitive(t *testing.T) {
	g, _ := fakeStripe(t, 400, `{"error": {"type": "invalid_request_error", "message": "Refund amount is greater than unrefunded amount"}}`)

	_, err := g.RefundPaymentIntent("pi_1", 1999, uuid.New())

	assert.ErrorIs(t, err, domain.ErrPaymentProviderRejected)
	assert.ErrorContains(t, err, "greater than unrefunded amount")
}

func TestRefundPaymentIntent_ServerErrorIsUnknownOutcome(t *testing.T) {
	g, _ := fakeStripe(t, 500, `{"error": {"type": "api_error", "message": "boom"}}`)

	_, err := g.RefundPaymentIntent("pi_1", 1999, uuid.New())

	assert.ErrorIs(t, err, domain.ErrPaymentProvider)
	assert.NotErrorIs(t, err, domain.ErrPaymentProviderRejected)
}

func TestGetPaymentMethodType_ExpandsLatestCharge(t *testing.T) {
	g, requests := fakeStripe(t, 200, `{"id": "pi_1", "object": "payment_intent",
		"latest_charge": {"id": "ch_1", "object": "charge", "payment_method_details": {"type": "boleto"}}}`)

	method, err := g.GetPaymentMethodType("pi_1")

	require.NoError(t, err)
	assert.Equal(t, "boleto", method)
	req := (*requests)[0]
	assert.Equal(t, "GET", req.method)
	assert.Equal(t, "/v1/payment_intents/pi_1", req.path)
	assert.Equal(t, "latest_charge", req.form.Get("expand[0]"))
}

func TestGetPaymentMethodType_WithoutCharge(t *testing.T) {
	g, _ := fakeStripe(t, 200, `{"id": "pi_1", "object": "payment_intent", "latest_charge": null}`)

	_, err := g.GetPaymentMethodType("pi_1")

	assert.ErrorIs(t, err, domain.ErrPaymentProvider)
}

func TestCancelPaymentIntent_UnexpectedState(t *testing.T) {
	g, requests := fakeStripe(t, 400, `{"error": {"type": "invalid_request_error", "code": "payment_intent_unexpected_state", "message": "cannot cancel"}}`)

	err := g.CancelPaymentIntent("pi_1")

	assert.ErrorIs(t, err, domain.ErrPaymentIntentUnexpectedState)
	assert.Equal(t, "/v1/payment_intents/pi_1/cancel", (*requests)[0].path)
}
