package service

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
)

type UserService interface {
	CreateUser(user *domain.User) error
	GetUserByID(id uuid.UUID) (*domain.User, error)
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

func (s *UserServiceImpl) GetUserByID(id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return user, nil
}
