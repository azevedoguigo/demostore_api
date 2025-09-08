package service

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
)

type ProductService interface {
	CreateProduct(dto request.CreateProductRequestDTO) error
	GetAllProducts() ([]domain.Product, error)
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
