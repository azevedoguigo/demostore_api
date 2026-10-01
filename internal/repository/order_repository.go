package repository

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func sortedByProduct(items []domain.OrderItem) []domain.OrderItem {
	sorted := append([]domain.OrderItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i].ProductID[:], sorted[j].ProductID[:]) < 0
	})

	return sorted
}

func (r *OrderRepository) CreateFromCart(order *domain.Order, cartID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range sortedByProduct(order.Items) {
			res := tx.Model(&domain.Product{}).
				Where("id = ? AND stock >= ?", item.ProductID, item.Quantity).
				UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("%w: %s", domain.ErrInsufficientStock, item.ProductName)
			}
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		return tx.Delete(&domain.CartItem{}, "cart_id = ?", cartID).Error
	})
}

func (r *OrderRepository) GetByID(id uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	err := r.db.Preload("Items").First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepository) GetByUserID(userID uuid.UUID) ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.Preload("Items").Where("user_id = ?", userID).Order("created_at DESC").Find(&orders).Error

	return orders, err
}

func (r *OrderRepository) GetAll() ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.Preload("Items").Order("created_at DESC").Find(&orders).Error

	return orders, err
}

func (r *OrderRepository) UpdateStatus(order *domain.Order, from domain.OrderStatus, restock bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.Order{}).
			Where("id = ? AND status = ?", order.ID, from).
			Update("status", order.Status)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrInvalidStatusTransition
		}

		if !restock {
			return nil
		}

		for _, item := range sortedByProduct(order.Items) {
			quantity := item.Quantity - item.RefundedQuantity
			if quantity <= 0 {
				continue
			}

			err := tx.Unscoped().Model(&domain.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *OrderRepository) GetExpiredPending(now time.Time, limit int) ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.Preload("Items").
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", domain.OrderStatusPending, now).
		Order("expires_at").
		Limit(limit).
		Find(&orders).Error

	return orders, err
}

func (r *OrderRepository) ExtendExpiration(orderID uuid.UUID, expiresAt time.Time) error {
	return r.db.Model(&domain.Order{}).
		Where("id = ? AND status = ? AND (expires_at IS NULL OR expires_at < ?)", orderID, domain.OrderStatusPending, expiresAt).
		Update("expires_at", expiresAt).Error
}
