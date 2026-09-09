package repository_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/google/uuid"
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

func (s *ProductRepositoryTestSuite) TestGetAllProducts_Success() {
	products := []domain.Product{
		{
			Name:        "Product 1",
			Description: "Description for product 1",
			Price:       10.0,
			Stock:       20,
		},
		{
			Name:        "Product 2",
			Description: "Description for product 2",
			Price:       15.0,
			Stock:       30,
		},
	}

	for i := range products {
		products[i].BindID()
		err := s.repo.Create(&products[i])
		assert.Nil(s.T(), err)
	}

	retrievedProducts, err := s.repo.GetAll()

	assert.Nil(s.T(), err)
	assert.Equal(
		s.T(),
		len(products),
		len(retrievedProducts),
		"Number of retrieved products should match the number of created products",
	)
}

func (s *ProductRepositoryTestSuite) TestGetAllProducts_Empty() {
	retrievedProducts, err := s.repo.GetAll()

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, len(retrievedProducts), "Retrieved products should be empty when no products exist")
}

func (s *ProductRepositoryTestSuite) TestGetProductByID_Success() {
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

	retrievedProduct, err := s.repo.GetByID(product.ID)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), product.ID, retrievedProduct.ID)
}

func (s *ProductRepositoryTestSuite) TestUpdateProduct_Success() {
	product := &domain.Product{
		Name:        "Test Product",
		Description: "This is a test product description.",
		Price:       19.99,
		Stock:       100,
	}
	product.BindID()
	err := s.repo.Create(product)
	assert.Nil(s.T(), err)

	product.Name = "Updated Product"
	product.Price = 29.99
	err = s.repo.Update(product)
	assert.Nil(s.T(), err)

	updated, err := s.repo.GetByID(product.ID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "Updated Product", updated.Name)
	assert.Equal(s.T(), 29.99, updated.Price)
}

func (s *ProductRepositoryTestSuite) TestDeleteProduct_Success() {
	product := &domain.Product{
		Name:        "Test Product",
		Description: "This is a test product description.",
		Price:       19.99,
		Stock:       100,
	}
	product.BindID()
	err := s.repo.Create(product)
	assert.Nil(s.T(), err)

	err = s.repo.Delete(product.ID)
	assert.Nil(s.T(), err)

	deleted, err := s.repo.GetByID(product.ID)
	assert.Nil(s.T(), deleted)
	assert.ErrorAs(s.T(), err, &gorm.ErrRecordNotFound)
}

func (s *ProductRepositoryTestSuite) TestGetProductByID_NotFound() {
	id := uuid.New()
	product, err := s.repo.GetByID(id)

	assert.Nil(s.T(), product, "Product should be nil when not found")
	assert.Error(s.T(), err, "Expected an error when product is not found")
	assert.ErrorAs(s.T(), err, &gorm.ErrRecordNotFound, "Expected gorm.ErrRecordNotFound error")
}

func TestProductRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ProductRepositoryTestSuite))
}
