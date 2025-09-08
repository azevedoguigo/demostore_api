package service_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Create(product *domain.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductRepository) GetAll() ([]domain.Product, error) {
	args := m.Called()
	return args.Get(0).([]domain.Product), args.Error(1)
}

func (m *MockProductRepository) GetByID(id uuid.UUID) (*domain.Product, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Product), args.Error(1)
}

type ProductServiceTestSuite struct {
	suite.Suite
	repo    *MockProductRepository
	service *service.ProductServiceImpl
}

func (suite *ProductServiceTestSuite) SetupTest() {
	suite.repo = new(MockProductRepository)
	suite.service = service.NewProductService(suite.repo)
}

func (suite *ProductServiceTestSuite) TestCreateProduct_Success() {
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "This is a test product",
		Price:       99.99,
		Stock:       10,
	}

	suite.repo.On("Create", mock.AnythingOfType("*domain.Product")).Return(nil)

	err := suite.service.CreateProduct(dto)

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestCreateProduct_RepositoryError() {
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "This is a test product",
		Price:       99.99,
		Stock:       10,
	}

	suite.repo.On("Create", mock.AnythingOfType("*domain.Product")).Return(assert.AnError)

	err := suite.service.CreateProduct(dto)

	suite.Error(err)
	suite.Equal(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetAllProducts_Success() {
	products := []domain.Product{
		{
			Name:        "Product 1",
			Description: "Description for product 1",
			Price:       10.0,
			Stock:       5,
		},
		{
			Name:        "Product 2",
			Description: "Description for product 2",
			Price:       20.0,
			Stock:       15,
		},
	}

	suite.repo.On("GetAll").Return(products, nil)

	result, err := suite.service.GetAllProducts()

	suite.NoError(err)
	suite.Equal(products, result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetAllProducts_Empty() {
	suite.repo.On("GetAll").Return([]domain.Product{}, nil)

	result, err := suite.service.GetAllProducts()

	suite.NoError(err)
	suite.Empty(result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetAllProducts_RepositoryError() {
	suite.repo.On("GetAll").Return([]domain.Product(nil), assert.AnError)

	result, err := suite.service.GetAllProducts()

	suite.Error(err)
	suite.Nil(result)
	suite.Equal(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetProductByID_Success() {
	productID := uuid.New()
	product := &domain.Product{
		ID:          productID,
		Name:        "Test Product",
		Description: "This is a test product description.",
		Price:       19.99,
		Stock:       100,
	}

	suite.repo.On("GetByID", productID).Return(product, nil)

	result, err := suite.service.GetProductByID(productID.String())

	suite.NoError(err)
	suite.Equal(product, result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetProductByID_NotFound() {
	productID := uuid.New()

	suite.repo.On("GetByID", productID).Return((*domain.Product)(nil), assert.AnError)

	result, err := suite.service.GetProductByID(productID.String())

	suite.Error(err)
	suite.Nil(result)
	suite.Equal(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func TestProductServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProductServiceTestSuite))
}
