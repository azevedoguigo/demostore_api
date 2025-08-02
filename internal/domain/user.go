package domain

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name     string    `json:"name" validate:"required,min=3,max=100"`
	Email    string    `gorm:"unique" json:"email" validate:"required,email"`
	Password string    `json:"password" validate:"required,min=6,max=100"`
}

type UserRepository interface {
	Create(user *User) error
	GetByID(id uuid.UUID) (*User, error)
}

func (u *User) BindID() {
	u.ID = uuid.New()
}

func (u *User) BindHashedPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)
	return nil
}
