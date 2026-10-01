package repository

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *domain.Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) GetByOrderID(orderID uuid.UUID) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.First(&payment, "order_id = ?", orderID).Error
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) GetByProviderPaymentID(providerPaymentID string) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.First(&payment, "provider_payment_id = ?", providerPaymentID).Error
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) Update(payment *domain.Payment) error {
	return r.db.Save(payment).Error
}
