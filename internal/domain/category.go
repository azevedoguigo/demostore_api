package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	gorm.Model  `swaggerignore:"true"`
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `json:"name" validate:"required,min=3,max=100"`
	Description string    `json:"description" validate:"required,min=10,max=500"`
}

type CategoryRepository interface {
	Create(category *Category) error
	GetAll() ([]Category, error)
	GetByID(id uuid.UUID) (*Category, error)
	Update(category *Category) error
	Delete(id uuid.UUID) error
}

func (c *Category) BindID() {
	c.ID = uuid.New()
}
