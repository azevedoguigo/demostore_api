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

const (
	pixMinAmount    int64 = 50
	pixMaxAmount    int64 = 300_000
	boletoMinAmount int64 = 500
	boletoMaxAmount int64 = 4_999_999
)

const BoletoConfirmationGrace = 5 * 24 * time.Hour

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

func RefundSupported(paymentMethodType string) bool {
	return paymentMethodType != PaymentMethodBoleto
}

var (
	ErrPaymentProvider              = errors.New("payment provider error")
	ErrPaymentProviderRejected      = errors.New("request rejected by the payment provider")
	ErrPaymentIntentUnexpectedState = errors.New("payment intent is in an unexpected state")
	ErrInvalidWebhookSignature      = errors.New("invalid webhook signature")
)

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

func (p *Payment) RemainingRefundable() int64 {
	return p.Amount - p.RefundedAmount
}

type PaymentIntentRequest struct {
	OrderID            uuid.UUID
	Amount             int64
	Currency           string
	PaymentMethodTypes []string
	ExpiresAt          time.Time
}

type PaymentIntent struct {
	ID           string
	ClientSecret string
	Status       string
}

const PaymentIntentStatusCanceled = "canceled"

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
	Type                PaymentEventType
	PaymentIntentID     string
	NextActionType      string
	NextActionExpiresAt time.Time
	Refund              *ProviderRefund
}

type ProviderRefund struct {
	ID            string
	Status        string
	LocalRefundID string
}

const (
	ProviderRefundSucceeded = "succeeded"
	ProviderRefundFailed    = "failed"
	ProviderRefundCanceled  = "canceled"
)

type PaymentGateway interface {
	CreatePaymentIntent(req PaymentIntentRequest) (*PaymentIntent, error)
	GetPaymentIntent(id string) (*PaymentIntent, error)
	CancelPaymentIntent(id string) error
	GetPaymentMethodType(paymentIntentID string) (string, error)
	RefundPaymentIntent(paymentIntentID string, amount int64, refundID uuid.UUID) (*ProviderRefund, error)
	ParseWebhookEvent(payload []byte, signature string) (*PaymentEvent, error)
}
