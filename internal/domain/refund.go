package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "pending"
	RefundStatusSucceeded RefundStatus = "succeeded"
	RefundStatusFailed    RefundStatus = "failed"
)

var ErrRefundQuantityExceeded = errors.New("refund quantity exceeds the quantity available to refund")

type Refund struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID          uuid.UUID    `gorm:"type:uuid;not null;index" json:"order_id"`
	PaymentID        uuid.UUID    `gorm:"type:uuid;not null;index" json:"payment_id"`
	ProviderRefundID *string      `gorm:"uniqueIndex" json:"provider_refund_id,omitempty"`
	Amount           int64        `gorm:"not null" json:"amount"`
	Reason           string       `json:"reason"`
	Status           RefundStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	Restock          bool         `gorm:"not null" json:"restock"`
	Items            []RefundItem `gorm:"foreignKey:RefundID" json:"items"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type RefundItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	RefundID    uuid.UUID `gorm:"type:uuid;not null;index" json:"refund_id"`
	OrderItemID uuid.UUID `gorm:"type:uuid;not null" json:"order_item_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Amount      int64     `gorm:"not null" json:"amount"`
}

type RefundRepository interface {
	Reserve(refund *Refund) error
	SetProviderRefundID(id uuid.UUID, providerRefundID string) error
	MarkSucceeded(refund *Refund) (bool, error)
	MarkFailed(refund *Refund) (bool, error)
	GetByID(id uuid.UUID) (*Refund, error)
	GetByProviderRefundID(providerRefundID string) (*Refund, error)
	GetByOrderID(orderID uuid.UUID) ([]Refund, error)
	GetPending(paymentID uuid.UUID) ([]Refund, error)
}

func (r *Refund) BindID() {
	r.ID = uuid.New()
}

func (i *RefundItem) BindID() {
	i.ID = uuid.New()
}
