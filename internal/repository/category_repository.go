package repository

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *domain.Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) GetAll() ([]domain.Category, error) {
	var categories []domain.Category
	err := r.db.Find(&categories).Error

	return categories, err
}

func (r *CategoryRepository) GetByID(id uuid.UUID) (*domain.Category, error) {
	var category domain.Category
	err := r.db.First(&category, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (r *CategoryRepository) Update(category *domain.Category) error {
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Category{}, "id = ?", id).Error
}
