package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const PaymentProviderStripe = "stripe"

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"

	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
)

const (
	PaymentMethodCard   = "card"
	PaymentMethodPix    = "pix"
	PaymentMethodBoleto = "boleto"
)

// Per-method amount limits in cents, as documented by Stripe for BRL.
const (
	pixMinAmount    int64 = 50
	pixMaxAmount    int64 = 300_000
	boletoMinAmount int64 = 500
	boletoMaxAmount int64 = 4_999_999
)

// BoletoConfirmationGrace is how long a pending order is kept after its boleto expires. Stripe
// confirms a paid boleto (or reports it unpaid) one business day later, so this covers weekends
// and holidays before the order is cancelled and its stock released.
const BoletoConfirmationGrace = 5 * 24 * time.Hour

// PaymentMethodTypesFor returns the payment methods that accept the given amount.
func PaymentMethodTypesFor(amount int64) []string {
	types := []string{PaymentMethodCard}
	if amount >= pixMinAmount && amount <= pixMaxAmount {
		types = append(types, PaymentMethodPix)
	}
	if amount >= boletoMinAmount && amount <= boletoMaxAmount {
		types = append(types, PaymentMethodBoleto)
	}

	return types
}

// RefundSupported reports whether payments made with the given method can be refunded through the
// provider. Stripe doesn't support refunds for boleto, which must be refunded outside of it.
func RefundSupported(paymentMethodType string) bool {
	return paymentMethodType != PaymentMethodBoleto
}

var (
	// ErrPaymentProvider wraps any failure talking to the payment provider.
	ErrPaymentProvider = errors.New("payment provider error")
	// ErrPaymentProviderRejected marks provider errors that are a definitive answer (the request was
	// refused), as opposed to network or server failures where the outcome is unknown.
	ErrPaymentProviderRejected = errors.New("request rejected by the payment provider")
	// ErrPaymentIntentUnexpectedState is returned when the provider refuses an operation
	// because the payment intent is no longer in a state that allows it (e.g. cancelling a paid intent).
	ErrPaymentIntentUnexpectedState = errors.New("payment intent is in an unexpected state")
	ErrInvalidWebhookSignature      = errors.New("invalid webhook signature")
)

// Payment links an order to its payment at the provider. Amount is in cents.
type Payment struct {
	ID                uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID           uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex" json:"order_id"`
	Provider          string        `gorm:"type:varchar(20);not null" json:"provider"`
	ProviderPaymentID string        `gorm:"not null;uniqueIndex" json:"provider_payment_id"`
	Status            PaymentStatus `gorm:"type:varchar(20);not null" json:"status"`
	Amount            int64         `gorm:"not null" json:"amount"`
	RefundedAmount    int64         `gorm:"not null;default:0" json:"refunded_amount"`
	Currency          string        `gorm:"type:varchar(3);not null" json:"currency"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type PaymentRepository interface {
	Create(payment *Payment) error
	GetByOrderID(orderID uuid.UUID) (*Payment, error)
	GetByProviderPaymentID(providerPaymentID string) (*Payment, error)
	Update(payment *Payment) error
}

func (p *Payment) BindID() {
	p.ID = uuid.New()
}

// RemainingRefundable is the paid amount not yet refunded.
func (p *Payment) RemainingRefundable() int64 {
	return p.Amount - p.RefundedAmount
}

type PaymentIntentRequest struct {
	OrderID            uuid.UUID
	Amount             int64
	Currency           string
	PaymentMethodTypes []string
	// ExpiresAt bounds how long a Pix QR code stays payable, aligned with the order expiration.
	ExpiresAt time.Time
}

type PaymentIntent struct {
	ID           string
	ClientSecret string
	Status       string
}

const PaymentIntentStatusCanceled = "canceled"

// NextActionBoletoDisplayDetails is the next action of a payment intent with an issued boleto.
const NextActionBoletoDisplayDetails = "boleto_display_details"

type PaymentEventType string

const (
	PaymentEventSucceeded      PaymentEventType = "payment_intent.succeeded"
	PaymentEventFailed         PaymentEventType = "payment_intent.payment_failed"
	PaymentEventCanceled       PaymentEventType = "payment_intent.canceled"
	PaymentEventRequiresAction PaymentEventType = "payment_intent.requires_action"
	PaymentEventRefundCreated  PaymentEventType = "refund.created"
	PaymentEventRefundUpdated  PaymentEventType = "refund.updated"
	PaymentEventRefundFailed   PaymentEventType = "refund.failed"
)

type PaymentEvent struct {
	Type PaymentEventType
	// Set for payment_intent.* events.
	PaymentIntentID     string
	NextActionType      string
	NextActionExpiresAt time.Time
	// Set for refund.* events.
	Refund *ProviderRefund
}

// ProviderRefund is a refund as reported by the provider. LocalRefundID is our Refund ID,
// sent as metadata so refunds can be matched even before their provider ID is stored.
type ProviderRefund struct {
	ID            string
	Status        string
	LocalRefundID string
}

// Provider refund statuses.
const (
	ProviderRefundSucceeded = "succeeded"
	ProviderRefundFailed    = "failed"
	ProviderRefundCanceled  = "canceled"
)

// PaymentGateway abstracts the payment provider (Stripe) so services can be tested without network calls.
type PaymentGateway interface {
	CreatePaymentIntent(req PaymentIntentRequest) (*PaymentIntent, error)
	GetPaymentIntent(id string) (*PaymentIntent, error)
	CancelPaymentIntent(id string) error
	// GetPaymentMethodType returns the method (card, pix, boleto...) used to pay a succeeded intent.
	GetPaymentMethodType(paymentIntentID string) (string, error)
	// RefundPaymentIntent refunds amount cents. refundID is used as idempotency key and metadata.
	RefundPaymentIntent(paymentIntentID string, amount int64, refundID uuid.UUID) (*ProviderRefund, error)
	// ParseWebhookEvent verifies the signature and returns the event. Events unrelated to payment
	// intents or refunds are returned with only their Type set.
	ParseWebhookEvent(payload []byte, signature string) (*PaymentEvent, error)
}
