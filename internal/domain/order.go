package domain

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// DefaultCurrency is the ISO currency code (lowercase, as Stripe expects) used for orders.
const DefaultCurrency = "brl"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

var orderTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending: {OrderStatusPaid, OrderStatusCancelled},
	OrderStatusPaid:    {OrderStatusShipped, OrderStatusCancelled},
	OrderStatusShipped: {OrderStatusDelivered},
}

var (
	ErrInsufficientStock       = errors.New("insufficient stock")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
)

// Order stores monetary values in the smallest currency unit (cents), matching Stripe's amount format.
type Order struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id"`
	Status      OrderStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	TotalAmount int64       `gorm:"not null" json:"total_amount"`
	Currency    string      `gorm:"type:varchar(3);not null" json:"currency"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// OrderItem snapshots the product name and price at checkout time, so later product changes don't alter the order.
type OrderItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductName string    `gorm:"not null" json:"product_name"`
	UnitPrice   int64     `gorm:"not null" json:"unit_price"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Subtotal    int64     `gorm:"not null" json:"subtotal"`
}

type OrderRepository interface {
	// CreateFromCart atomically decrements stock for every item, persists the order and empties the cart.
	CreateFromCart(order *Order, cartID uuid.UUID) error
	GetByID(id uuid.UUID) (*Order, error)
	GetByUserID(userID uuid.UUID) ([]Order, error)
	GetAll() ([]Order, error)
	// UpdateStatus persists order.Status only if the stored status is still `from`, restoring stock when restock is true.
	UpdateStatus(order *Order, from OrderStatus, restock bool) error
}

func (o *Order) BindID() {
	o.ID = uuid.New()
}

func (i *OrderItem) BindID() {
	i.ID = uuid.New()
}

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusPaid, OrderStatusShipped, OrderStatusDelivered, OrderStatusCancelled:
		return true
	}

	return false
}

func (o *Order) CanTransitionTo(next OrderStatus) bool {
	for _, allowed := range orderTransitions[o.Status] {
		if allowed == next {
			return true
		}
	}

	return false
}

// ToCents converts a decimal price to the smallest currency unit.
func ToCents(price float64) int64 {
	return int64(math.Round(price * 100))
}
