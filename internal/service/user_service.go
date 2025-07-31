package service

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: *repo}
}

func (s *UserService) CreateUser(user *domain.User) error {
	if err := s.repo.Create(user); err != nil {
		return err
	}
	return nil
}
