package repository

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) *CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) Create(cart *domain.Cart) error {
	return r.db.Create(cart).Error
}

func (r *CartRepository) GetByUserID(userID uuid.UUID) (*domain.Cart, error) {
	var cart domain.Cart
	err := r.db.Preload("Items.Product").First(&cart, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}

	return &cart, nil
}

func (r *CartRepository) GetItem(cartID, productID uuid.UUID) (*domain.CartItem, error) {
	var item domain.CartItem
	err := r.db.First(&item, "cart_id = ? AND product_id = ?", cartID, productID).Error
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *CartRepository) SaveItem(item *domain.CartItem) error {
	return r.db.Save(item).Error
}

func (r *CartRepository) DeleteItem(cartID, productID uuid.UUID) error {
	return r.db.Delete(&domain.CartItem{}, "cart_id = ? AND product_id = ?", cartID, productID).Error
}

func (r *CartRepository) Clear(cartID uuid.UUID) error {
	return r.db.Delete(&domain.CartItem{}, "cart_id = ?", cartID).Error
}
