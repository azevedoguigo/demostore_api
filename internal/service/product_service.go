package service

import (
	"errors"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrCategoryNotFound = errors.New("category not found")

type ProductService interface {
	CreateProduct(dto request.CreateProductRequestDTO) error
	GetAllProducts() ([]domain.Product, error)
	GetProductByID(id string) (*domain.Product, error)
	UpdateProduct(id string, dto request.UpdateProductRequestDTO) error
	DeleteProduct(id string) error
}

type ProductServiceImpl struct {
	repo         domain.ProductRepository
	categoryRepo domain.CategoryRepository
}

func NewProductService(repo domain.ProductRepository, categoryRepo domain.CategoryRepository) *ProductServiceImpl {
	return &ProductServiceImpl{repo: repo, categoryRepo: categoryRepo}
}

func (s *ProductServiceImpl) CreateProduct(dto request.CreateProductRequestDTO) error {
	categoryID, err := uuid.Parse(dto.CategoryID)
	if err != nil {
		return err
	}

	if _, err := s.categoryRepo.GetByID(categoryID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCategoryNotFound
		}

		return err
	}

	product := &domain.Product{
		Name:        dto.Name,
		Description: dto.Description,
		Price:       dto.Price,
		Stock:       dto.Stock,
		CategoryID:  categoryID,
	}

	product.BindID()

	if err := s.repo.Create(product); err != nil {
		return err
	}

	return nil
}

func (s *ProductServiceImpl) GetAllProducts() ([]domain.Product, error) {
	return s.repo.GetAll()
}

func (s *ProductServiceImpl) GetProductByID(id string) (*domain.Product, error) {
	productID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}
	return product, nil
}

func (s *ProductServiceImpl) UpdateProduct(id string, dto request.UpdateProductRequestDTO) error {
	productID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	categoryID, err := uuid.Parse(dto.CategoryID)
	if err != nil {
		return err
	}

	if _, err := s.categoryRepo.GetByID(categoryID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCategoryNotFound
		}

		return err
	}

	product, err := s.repo.GetByID(productID)
	if err != nil {
		return err
	}

	product.Name = dto.Name
	product.Description = dto.Description
	product.Price = dto.Price
	product.Stock = dto.Stock
	product.CategoryID = categoryID

	return s.repo.Update(product)
}

func (s *ProductServiceImpl) DeleteProduct(id string) error {
	productID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	if _, err := s.repo.GetByID(productID); err != nil {
		return err
	}

	return s.repo.Delete(productID)
}
