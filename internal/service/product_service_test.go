package service_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
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

func TestProductServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProductServiceTestSuite))
}
