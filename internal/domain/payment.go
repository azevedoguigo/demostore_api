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
)

var (
	// ErrPaymentProvider wraps any failure talking to the payment provider.
	ErrPaymentProvider = errors.New("payment provider error")
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

type PaymentIntent struct {
	ID           string
	ClientSecret string
	Status       string
}

type PaymentEventType string

const (
	PaymentEventSucceeded PaymentEventType = "payment_intent.succeeded"
	PaymentEventFailed    PaymentEventType = "payment_intent.payment_failed"
	PaymentEventCanceled  PaymentEventType = "payment_intent.canceled"
)

type PaymentEvent struct {
	Type            PaymentEventType
	PaymentIntentID string
}

// PaymentGateway abstracts the payment provider (Stripe) so services can be tested without network calls.
type PaymentGateway interface {
	CreatePaymentIntent(orderID uuid.UUID, amount int64, currency string) (*PaymentIntent, error)
	GetPaymentIntent(id string) (*PaymentIntent, error)
	CancelPaymentIntent(id string) error
	RefundPaymentIntent(id string) error
	// ParseWebhookEvent verifies the signature and returns the event. Events unrelated to
	// payment intents are returned with an empty PaymentIntentID.
	ParseWebhookEvent(payload []byte, signature string) (*PaymentEvent, error)
}
