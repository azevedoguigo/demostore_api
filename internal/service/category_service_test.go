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

type CategoryServiceTestSuite struct {
	suite.Suite
	repo    *MockCategoryRepository
	service *service.CategoryServiceImpl
}

func (suite *CategoryServiceTestSuite) SetupTest() {
	suite.repo = new(MockCategoryRepository)
	suite.service = service.NewCategoryService(suite.repo)
}

func (suite *CategoryServiceTestSuite) TestCreateCategory_Success() {
	dto := request.CreateCategoryRequestDTO{
		Name:        "Test Category",
		Description: "This is a test category",
	}

	suite.repo.On("Create", mock.AnythingOfType("*domain.Category")).Return(nil)

	err := suite.service.CreateCategory(dto)

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestCreateCategory_RepositoryError() {
	dto := request.CreateCategoryRequestDTO{
		Name:        "Test Category",
		Description: "This is a test category",
	}

	suite.repo.On("Create", mock.AnythingOfType("*domain.Category")).Return(assert.AnError)

	err := suite.service.CreateCategory(dto)

	suite.Error(err)
	suite.Equal(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetAllCategories_Success() {
	categories := []domain.Category{
		{Name: "Category 1", Description: "Description for category 1"},
		{Name: "Category 2", Description: "Description for category 2"},
	}

	suite.repo.On("GetAll").Return(categories, nil)

	result, err := suite.service.GetAllCategories()

	suite.NoError(err)
	suite.Equal(categories, result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetAllCategories_RepositoryError() {
	suite.repo.On("GetAll").Return([]domain.Category(nil), assert.AnError)

	result, err := suite.service.GetAllCategories()

	suite.Error(err)
	suite.Nil(result)
	suite.Equal(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetCategoryByID_Success() {
	categoryID := uuid.New()
	category := &domain.Category{
		ID:          categoryID,
		Name:        "Test Category",
		Description: "This is a test category description.",
	}

	suite.repo.On("GetByID", categoryID).Return(category, nil)

	result, err := suite.service.GetCategoryByID(categoryID.String())

	suite.NoError(err)
	suite.Equal(category, result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetCategoryByID_NotFound() {
	categoryID := uuid.New()

	suite.repo.On("GetByID", categoryID).Return((*domain.Category)(nil), gorm.ErrRecordNotFound)

	result, err := suite.service.GetCategoryByID(categoryID.String())

	suite.NoError(err)
	suite.Nil(result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetCategoryByID_InvalidID() {
	result, err := suite.service.GetCategoryByID("not-a-uuid")

	suite.Error(err)
	suite.Nil(result)
}

func (suite *CategoryServiceTestSuite) TestUpdateCategory_Success() {
	categoryID := uuid.New()
	existing := &domain.Category{
		ID:          categoryID,
		Name:        "Old Name",
		Description: "Old description",
	}
	dto := request.UpdateCategoryRequestDTO{
		Name:        "New Name",
		Description: "New description",
	}

	suite.repo.On("GetByID", categoryID).Return(existing, nil).Once()
	suite.repo.On("Update", mock.MatchedBy(func(c *domain.Category) bool {
		return c.Name == dto.Name && c.Description == dto.Description
	})).Return(nil).Once()

	err := suite.service.UpdateCategory(categoryID.String(), dto)

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestUpdateCategory_NotFound() {
	categoryID := uuid.New()
	dto := request.UpdateCategoryRequestDTO{
		Name:        "New Name",
		Description: "New description",
	}

	suite.repo.On("GetByID", categoryID).Return((*domain.Category)(nil), gorm.ErrRecordNotFound).Once()

	err := suite.service.UpdateCategory(categoryID.String(), dto)

	suite.ErrorIs(err, gorm.ErrRecordNotFound)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestUpdateCategory_RepositoryError() {
	categoryID := uuid.New()
	existing := &domain.Category{ID: categoryID}
	dto := request.UpdateCategoryRequestDTO{
		Name:        "New Name",
		Description: "New description",
	}

	suite.repo.On("GetByID", categoryID).Return(existing, nil).Once()
	suite.repo.On("Update", mock.AnythingOfType("*domain.Category")).Return(assert.AnError).Once()

	err := suite.service.UpdateCategory(categoryID.String(), dto)

	suite.ErrorIs(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestDeleteCategory_Success() {
	categoryID := uuid.New()
	existing := &domain.Category{ID: categoryID}

	suite.repo.On("GetByID", categoryID).Return(existing, nil).Once()
	suite.repo.On("Delete", categoryID).Return(nil).Once()

	err := suite.service.DeleteCategory(categoryID.String())

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestDeleteCategory_NotFound() {
	categoryID := uuid.New()

	suite.repo.On("GetByID", categoryID).Return((*domain.Category)(nil), gorm.ErrRecordNotFound).Once()

	err := suite.service.DeleteCategory(categoryID.String())

	suite.ErrorIs(err, gorm.ErrRecordNotFound)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestDeleteCategory_RepositoryError() {
	categoryID := uuid.New()
	existing := &domain.Category{ID: categoryID}

	suite.repo.On("GetByID", categoryID).Return(existing, nil).Once()
	suite.repo.On("Delete", categoryID).Return(assert.AnError).Once()

	err := suite.service.DeleteCategory(categoryID.String())

	suite.ErrorIs(err, assert.AnError)
	suite.repo.AssertExpectations(suite.T())
}

func TestCategoryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CategoryServiceTestSuite))
}
