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
	"gorm.io/gorm"
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
	p := args.Get(0)
	if p == nil {
		return nil, args.Error(1)
	}
	return p.(*domain.Product), args.Error(1)
}

func (m *MockProductRepository) Update(product *domain.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(category *domain.Category) error {
	args := m.Called(category)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetAll() ([]domain.Category, error) {
	args := m.Called()
	return args.Get(0).([]domain.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetByID(id uuid.UUID) (*domain.Category, error) {
	args := m.Called(id)
	c := args.Get(0)
	if c == nil {
		return nil, args.Error(1)
	}
	return c.(*domain.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(category *domain.Category) error {
	args := m.Called(category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

type ProductServiceTestSuite struct {
	suite.Suite
	repo         *MockProductRepository
	categoryRepo *MockCategoryRepository
	service      *service.ProductServiceImpl
}

func (suite *ProductServiceTestSuite) SetupTest() {
	suite.repo = new(MockProductRepository)
	suite.categoryRepo = new(MockCategoryRepository)
	suite.service = service.NewProductService(suite.repo, suite.categoryRepo)
}

func (suite *ProductServiceTestSuite) TestCreateProduct_Success() {
	categoryID := uuid.New()
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "This is a test product",
		Price:       99.99,
		Stock:       10,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return(&domain.Category{ID: categoryID}, nil).Once()
	suite.repo.On("Create", mock.AnythingOfType("*domain.Product")).Return(nil)

	err := suite.service.CreateProduct(dto)

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
	suite.categoryRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestCreateProduct_RepositoryError() {
	categoryID := uuid.New()
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "This is a test product",
		Price:       99.99,
		Stock:       10,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return(&domain.Category{ID: categoryID}, nil).Once()
	suite.repo.On("Create", mock.AnythingOfType("*domain.Product")).Return(assert.AnError)

	err := suite.service.CreateProduct(dto)

	suite.Error(err)
	suite.Equal(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestCreateProduct_CategoryNotFound() {
	categoryID := uuid.New()
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "This is a test product",
		Price:       99.99,
		Stock:       10,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return((*domain.Category)(nil), gorm.ErrRecordNotFound).Once()

	err := suite.service.CreateProduct(dto)

	suite.ErrorIs(err, service.ErrCategoryNotFound)
	suite.categoryRepo.AssertExpectations(suite.T())
	suite.repo.AssertNotCalled(suite.T(), "Create", mock.Anything)
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

func (suite *ProductServiceTestSuite) TestUpdateProduct_Success() {
	productID := uuid.New()
	categoryID := uuid.New()
	existing := &domain.Product{
		ID:          productID,
		Name:        "Old Name",
		Description: "Old description",
		Price:       10.0,
		Stock:       5,
	}
	dto := request.UpdateProductRequestDTO{
		Name:        "New Name",
		Description: "New description",
		Price:       20.0,
		Stock:       15,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return(&domain.Category{ID: categoryID}, nil).Once()
	suite.repo.On("GetByID", productID).Return(existing, nil).Once()
	suite.repo.On("Update", mock.MatchedBy(func(p *domain.Product) bool {
		return p.Name == dto.Name && p.Description == dto.Description &&
			p.Price == dto.Price && p.Stock == dto.Stock && p.CategoryID == categoryID
	})).Return(nil).Once()

	err := suite.service.UpdateProduct(productID.String(), dto)

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
	suite.categoryRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestUpdateProduct_NotFound() {
	productID := uuid.New()
	categoryID := uuid.New()
	dto := request.UpdateProductRequestDTO{
		Name:        "New Name",
		Description: "New description",
		Price:       20.0,
		Stock:       15,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return(&domain.Category{ID: categoryID}, nil).Once()
	suite.repo.On("GetByID", productID).Return((*domain.Product)(nil), gorm.ErrRecordNotFound).Once()

	err := suite.service.UpdateProduct(productID.String(), dto)

	suite.ErrorIs(err, gorm.ErrRecordNotFound)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestUpdateProduct_CategoryNotFound() {
	productID := uuid.New()
	categoryID := uuid.New()
	dto := request.UpdateProductRequestDTO{
		Name:        "New Name",
		Description: "New description",
		Price:       20.0,
		Stock:       15,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return((*domain.Category)(nil), gorm.ErrRecordNotFound).Once()

	err := suite.service.UpdateProduct(productID.String(), dto)

	suite.ErrorIs(err, service.ErrCategoryNotFound)
	suite.categoryRepo.AssertExpectations(suite.T())
	suite.repo.AssertNotCalled(suite.T(), "GetByID", mock.Anything)
}

func (suite *ProductServiceTestSuite) TestUpdateProduct_RepositoryError() {
	productID := uuid.New()
	categoryID := uuid.New()
	existing := &domain.Product{ID: productID}
	dto := request.UpdateProductRequestDTO{
		Name:        "New Name",
		Description: "New description",
		Price:       20.0,
		Stock:       15,
		CategoryID:  categoryID.String(),
	}

	suite.categoryRepo.On("GetByID", categoryID).Return(&domain.Category{ID: categoryID}, nil).Once()
	suite.repo.On("GetByID", productID).Return(existing, nil).Once()
	suite.repo.On("Update", mock.AnythingOfType("*domain.Product")).Return(assert.AnError).Once()

	err := suite.service.UpdateProduct(productID.String(), dto)

	suite.ErrorIs(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestDeleteProduct_Success() {
	productID := uuid.New()
	existing := &domain.Product{ID: productID}

	suite.repo.On("GetByID", productID).Return(existing, nil).Once()
	suite.repo.On("Delete", productID).Return(nil).Once()

	err := suite.service.DeleteProduct(productID.String())

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestDeleteProduct_NotFound() {
	productID := uuid.New()

	suite.repo.On("GetByID", productID).Return((*domain.Product)(nil), gorm.ErrRecordNotFound).Once()

	err := suite.service.DeleteProduct(productID.String())

	suite.ErrorIs(err, gorm.ErrRecordNotFound)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestDeleteProduct_RepositoryError() {
	productID := uuid.New()
	existing := &domain.Product{ID: productID}

	suite.repo.On("GetByID", productID).Return(existing, nil).Once()
	suite.repo.On("Delete", productID).Return(assert.AnError).Once()

	err := suite.service.DeleteProduct(productID.String())

	suite.ErrorIs(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func TestProductServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProductServiceTestSuite))
}
