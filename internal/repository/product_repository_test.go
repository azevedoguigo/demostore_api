package repository_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ProductRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *repository.ProductRepository
}

func (s *ProductRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	db.AutoMigrate(&domain.Product{})
	s.repo = repository.NewProductRepository(db)
}

func (s *ProductRepositoryTestSuite) TestCreateProduct_Success() {
	product := &domain.Product{
		Name:        "Test Product",
		Description: "This is a test product description.",
		Price:       19.99,
		Stock:       100,
	}
	product.BindID()
	err := s.repo.Create(product)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), product.ID, "Product ID should be set after creation")
}

func (s *ProductRepositoryTestSuite) TestCreateProduct_Error() {
	product := &domain.Product{
		Name:        "Test Product",
		Description: "This is a test product description.",
		Price:       19.99,
		Stock:       100,
	}
	product.BindID()
	err := s.repo.Create(product)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), product.ID, "Product ID should be set after creation")

	productWithSameID := &domain.Product{
		ID:          product.ID,
		Name:        "Another Product",
		Description: "This is another product description.",
		Price:       29.99,
		Stock:       50,
	}
	err = s.repo.Create(productWithSameID)

	assert.NotNil(s.T(), err, "Creating a product with duplicate ID should return an error")
}

func TestProductRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ProductRepositoryTestSuite))
}
