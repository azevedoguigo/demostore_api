package service

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
)

type UserService interface {
	CreateUser(user *domain.User) error
	GetUserByID(id uuid.UUID) (*domain.User, error)
	GetUserByEmail(email string) (*domain.User, error)
	UpdateUser(dto *request.UpdateUserRequestDTO) error
}

type UserServiceImpl struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{repo: repo}
}

func (s *UserServiceImpl) CreateUser(user *domain.User) error {
	user.BindID()

	if err := user.BindHashedPassword(user.Password); err != nil {
		return err
	}

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

func (s *UserServiceImpl) GetUserByEmail(email string) (*domain.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) UpdateUser(dto *request.UpdateUserRequestDTO) error {
	uuid, err := uuid.Parse(dto.ID)
	if err != nil {
		return err
	}

	user, err := s.repo.GetByID(uuid)
	if err != nil {
		return err
	}

	if dto.Name != "" {
		user.Name = dto.Name
	}

	if dto.AccessToken != "" {
		user.AccessToken = dto.AccessToken
	}

	if err := s.repo.Update(user); err != nil {
		return err
	}

	return nil
}
