package repository

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(product *domain.Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) GetAll() ([]domain.Product, error) {
	var products []domain.Product
	err := r.db.Find(&products).Error

	return products, err
}

func (r *ProductRepository) GetByID(id uuid.UUID) (*domain.Product, error) {
	var product domain.Product
	err := r.db.First(&product, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &product, nil
}
