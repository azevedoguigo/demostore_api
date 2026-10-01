package domain_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPaymentMethodTypesFor(t *testing.T) {
	cases := map[int64][]string{
		50:        {"card", "pix"},
		499:       {"card", "pix"},
		500:       {"card", "pix", "boleto"},
		300_000:   {"card", "pix", "boleto"},
		300_001:   {"card", "boleto"},
		4_999_999: {"card", "boleto"},
		5_000_000: {"card"},
	}

	for amount, expected := range cases {
		assert.Equal(t, expected, domain.PaymentMethodTypesFor(amount), amount)
	}
}

func TestRefundSupported(t *testing.T) {
	assert.True(t, domain.RefundSupported(domain.PaymentMethodCard))
	assert.True(t, domain.RefundSupported(domain.PaymentMethodPix))
	assert.False(t, domain.RefundSupported(domain.PaymentMethodBoleto))
}

func TestPayment_RemainingRefundable(t *testing.T) {
	payment := domain.Payment{Amount: 4028, RefundedAmount: 1999}

	assert.Equal(t, int64(2029), payment.RemainingRefundable())
}
