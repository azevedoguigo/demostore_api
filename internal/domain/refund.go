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

// Refund tracks a (partial or full) refund of an order payment. While pending, the refunded
// quantities of its items are reserved on the order items so concurrent refunds can't exceed them.
type Refund struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID          uuid.UUID    `gorm:"type:uuid;not null;index" json:"order_id"`
	PaymentID        uuid.UUID    `gorm:"type:uuid;not null;index" json:"payment_id"`
	ProviderRefundID *string      `gorm:"uniqueIndex" json:"provider_refund_id,omitempty"`
	Amount           int64        `gorm:"not null" json:"amount"`
	Reason           string       `json:"reason"`
	Status           RefundStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	// Restock returns the refunded items to stock once the refund succeeds. It is false for the
	// refund issued when a paid order is cancelled, since cancelling the order already restocks.
	Restock   bool         `gorm:"not null" json:"restock"`
	Items     []RefundItem `gorm:"foreignKey:RefundID" json:"items"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
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
	// Reserve stores a pending refund and atomically reserves its items' quantities,
	// returning ErrRefundQuantityExceeded when an item has fewer units left to refund.
	Reserve(refund *Refund) error
	SetProviderRefundID(id uuid.UUID, providerRefundID string) error
	// MarkSucceeded settles a pending refund: adds its amount to the payment and restocks its items
	// when Restock is set. It returns false without changes if the refund was no longer pending.
	MarkSucceeded(refund *Refund) (bool, error)
	// MarkFailed releases the reserved quantities of a pending refund. It returns false without
	// changes if the refund was no longer pending.
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
