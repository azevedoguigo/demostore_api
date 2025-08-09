package service

import (
	"errors"

	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Login(email, password string) (string, error)
}

type AuthServiceImpl struct {
	userService UserService
}

func NewAuthService(userService UserService) *AuthServiceImpl {
	return &AuthServiceImpl{userService: userService}
}

func (s *AuthServiceImpl) Login(email, password string) (string, error) {
	user, err := s.userService.GetUserByEmail(email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", err
		}

		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid password")
	}

	err = user.BindAccessToken()
	if err != nil {
		return "", err
	}

	err = s.userService.UpdateUser(&request.UpdateUserRequestDTO{
		ID:          user.ID.String(),
		AccessToken: user.AccessToken,
	})
	if err != nil {
		return "", err
	}

	return user.AccessToken, nil
}
