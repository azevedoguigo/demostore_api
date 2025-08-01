package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `json:"name" validate:"required,min=3,max=100"`
	Email     string    `json:"email" validate:"required,email"`
	Password  string    `json:"password" validate:"required,min=6,max=100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRepository interface {
	Create(user *User) error
}

func (u *User) BindID() {
	u.ID = uuid.New()
}
