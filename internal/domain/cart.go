package domain

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Items     []CartItem `gorm:"foreignKey:CartID" json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CartID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_cart_product" json:"cart_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_cart_product" json:"product_id"`
	Product   *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CartRepository interface {
	Create(cart *Cart) error
	GetByUserID(userID uuid.UUID) (*Cart, error)
	GetItem(cartID, productID uuid.UUID) (*CartItem, error)
	SaveItem(item *CartItem) error
	DeleteItem(cartID, productID uuid.UUID) error
	Clear(cartID uuid.UUID) error
}

func (c *Cart) BindID() {
	c.ID = uuid.New()
}

func (i *CartItem) BindID() {
	i.ID = uuid.New()
}
