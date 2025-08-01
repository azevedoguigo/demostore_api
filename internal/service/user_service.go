package service

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
)

type UserService interface {
	CreateUser(user *domain.User) error
}

type UserServiceImpl struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{repo: repo}
}

func (s *UserServiceImpl) CreateUser(user *domain.User) error {
	user.BindID()

	if err := s.repo.Create(user); err != nil {
		return err
	}
	return nil
}
