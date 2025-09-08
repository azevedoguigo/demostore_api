package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `json:"name" validate:"required,min=3,max=100"`
	Description string    `json:"description" validate:"required,min=10,max=500"`
	Price       float64   `json:"price" validate:"required,gt=0"`
	Stock       int       `json:"stock" validate:"required,gte=0"`
}

type ProductRepository interface {
	Create(product *Product) error
	GetAll() ([]Product, error)
	GetByID(id uuid.UUID) (*Product, error)
}

func (p *Product) BindID() {
	p.ID = uuid.New()
}
