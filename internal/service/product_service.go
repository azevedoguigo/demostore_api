package service

import (
	"errors"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductService interface {
	CreateProduct(dto request.CreateProductRequestDTO) error
	GetAllProducts() ([]domain.Product, error)
	GetProductByID(id string) (*domain.Product, error)
}

type ProductServiceImpl struct {
	repo domain.ProductRepository
}

func NewProductService(repo domain.ProductRepository) *ProductServiceImpl {
	return &ProductServiceImpl{repo: repo}
}

func (s *ProductServiceImpl) CreateProduct(dto request.CreateProductRequestDTO) error {
	product := &domain.Product{
		Name:        dto.Name,
		Description: dto.Description,
		Price:       dto.Price,
		Stock:       dto.Stock,
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
	product, err := s.repo.GetByID(uuid.MustParse(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}
	return product, nil
}
