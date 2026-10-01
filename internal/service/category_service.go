package service

import (
	"errors"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryService interface {
	CreateCategory(dto request.CreateCategoryRequestDTO) error
	GetAllCategories() ([]domain.Category, error)
	GetCategoryByID(id string) (*domain.Category, error)
	UpdateCategory(id string, dto request.UpdateCategoryRequestDTO) error
	DeleteCategory(id string) error
}

type CategoryServiceImpl struct {
	repo domain.CategoryRepository
}

func NewCategoryService(repo domain.CategoryRepository) *CategoryServiceImpl {
	return &CategoryServiceImpl{repo: repo}
}

func (s *CategoryServiceImpl) CreateCategory(dto request.CreateCategoryRequestDTO) error {
	category := &domain.Category{
		Name:        dto.Name,
		Description: dto.Description,
	}

	category.BindID()

	return s.repo.Create(category)
}

func (s *CategoryServiceImpl) GetAllCategories() ([]domain.Category, error) {
	return s.repo.GetAll()
}

func (s *CategoryServiceImpl) GetCategoryByID(id string) (*domain.Category, error) {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	category, err := s.repo.GetByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return category, nil
}

func (s *CategoryServiceImpl) UpdateCategory(id string, dto request.UpdateCategoryRequestDTO) error {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	category, err := s.repo.GetByID(categoryID)
	if err != nil {
		return err
	}

	category.Name = dto.Name
	category.Description = dto.Description

	return s.repo.Update(category)
}

func (s *CategoryServiceImpl) DeleteCategory(id string) error {
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	if _, err := s.repo.GetByID(categoryID); err != nil {
		return err
	}

	return s.repo.Delete(categoryID)
}
