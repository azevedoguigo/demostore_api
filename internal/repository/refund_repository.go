package repository

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefundRepository struct {
	db *gorm.DB
}

func NewRefundRepository(db *gorm.DB) *RefundRepository {
	return &RefundRepository{db: db}
}

func (r *RefundRepository) Reserve(refund *domain.Refund) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range refund.Items {
			res := tx.Model(&domain.OrderItem{}).
				Where("id = ? AND order_id = ? AND refunded_quantity + ? <= quantity", item.OrderItemID, refund.OrderID, item.Quantity).
				UpdateColumn("refunded_quantity", gorm.Expr("refunded_quantity + ?", item.Quantity))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return domain.ErrRefundQuantityExceeded
			}
		}

		return tx.Create(refund).Error
	})
}

func (r *RefundRepository) SetProviderRefundID(id uuid.UUID, providerRefundID string) error {
	return r.db.Model(&domain.Refund{}).Where("id = ?", id).Update("provider_refund_id", providerRefundID).Error
}

func (r *RefundRepository) settle(refund *domain.Refund, status domain.RefundStatus, apply func(tx *gorm.DB) error) (bool, error) {
	settled := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.Refund{}).
			Where("id = ? AND status = ?", refund.ID, domain.RefundStatusPending).
			Update("status", status)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}

		settled = true
		return apply(tx)
	})
	if err != nil {
		return false, err
	}

	if settled {
		refund.Status = status
	}
	return settled, nil
}

func (r *RefundRepository) MarkSucceeded(refund *domain.Refund) (bool, error) {
	return r.settle(refund, domain.RefundStatusSucceeded, func(tx *gorm.DB) error {
		err := tx.Model(&domain.Payment{}).Where("id = ?", refund.PaymentID).Updates(map[string]interface{}{
			"refunded_amount": gorm.Expr("refunded_amount + ?", refund.Amount),
			"status": gorm.Expr("CASE WHEN refunded_amount + ? >= amount THEN ? ELSE ? END",
				refund.Amount, domain.PaymentStatusRefunded, domain.PaymentStatusPartiallyRefunded),
		}).Error
		if err != nil {
			return err
		}

		if !refund.Restock {
			return nil
		}

		for _, item := range refund.Items {
			err := tx.Unscoped().Model(&domain.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("stock", gorm.Expr("stock + ?", item.Quantity)).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *RefundRepository) MarkFailed(refund *domain.Refund) (bool, error) {
	return r.settle(refund, domain.RefundStatusFailed, func(tx *gorm.DB) error {
		for _, item := range refund.Items {
			err := tx.Model(&domain.OrderItem{}).
				Where("id = ?", item.OrderItemID).
				UpdateColumn("refunded_quantity", gorm.Expr("refunded_quantity - ?", item.Quantity)).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *RefundRepository) GetByID(id uuid.UUID) (*domain.Refund, error) {
	var refund domain.Refund
	err := r.db.Preload("Items").First(&refund, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &refund, nil
}

func (r *RefundRepository) GetByProviderRefundID(providerRefundID string) (*domain.Refund, error) {
	var refund domain.Refund
	err := r.db.Preload("Items").First(&refund, "provider_refund_id = ?", providerRefundID).Error
	if err != nil {
		return nil, err
	}

	return &refund, nil
}

func (r *RefundRepository) GetByOrderID(orderID uuid.UUID) ([]domain.Refund, error) {
	var refunds []domain.Refund
	err := r.db.Preload("Items").Where("order_id = ?", orderID).Order("created_at").Find(&refunds).Error

	return refunds, err
}

func (r *RefundRepository) GetPending(paymentID uuid.UUID) ([]domain.Refund, error) {
	var refunds []domain.Refund
	err := r.db.Preload("Items").
		Where("payment_id = ? AND status = ?", paymentID, domain.RefundStatusPending).
		Order("created_at").
		Find(&refunds).Error

	return refunds, err
}
