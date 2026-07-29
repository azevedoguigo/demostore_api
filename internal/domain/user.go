package domain

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model  `swaggerignore:"true"`
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `json:"name" validate:"required,min=3,max=100"`
	Email       string    `gorm:"unique" json:"email" validate:"required,email"`
	Password    string    `json:"password" validate:"required,min=6,max=100"`
	AccessToken string    `json:"access_token,omitempty"`
}

type UserRepository interface {
	Create(user *User) error
	GetByID(id uuid.UUID) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
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

func (u *User) BindAccessToken() error {
	var err error

	claims := jwt.StandardClaims{
		Id:        u.ID.String(),
		Subject:   u.Email,
		ExpiresAt: jwt.TimeFunc().Add(8 * time.Hour).Unix(),
		IssuedAt:  jwt.TimeFunc().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		return err
	}

	u.AccessToken = tokenString
	return nil
}
