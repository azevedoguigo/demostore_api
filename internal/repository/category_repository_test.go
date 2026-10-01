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

type CategoryRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *repository.CategoryRepository
}

func (s *CategoryRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	db.AutoMigrate(&domain.Category{})
	s.repo = repository.NewCategoryRepository(db)
}

func (s *CategoryRepositoryTestSuite) TestCreateCategory_Success() {
	category := &domain.Category{
		Name:        "Test Category",
		Description: "This is a test category description.",
	}
	category.BindID()
	err := s.repo.Create(category)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), category.ID, "Category ID should be set after creation")
}

func (s *CategoryRepositoryTestSuite) TestCreateCategory_Error() {
	category := &domain.Category{
		Name:        "Test Category",
		Description: "This is a test category description.",
	}
	category.BindID()
	err := s.repo.Create(category)
	assert.Nil(s.T(), err)

	categoryWithSameID := &domain.Category{
		ID:          category.ID,
		Name:        "Another Category",
		Description: "This is another category description.",
	}
	err = s.repo.Create(categoryWithSameID)

	assert.NotNil(s.T(), err, "Creating a category with duplicate ID should return an error")
}

func (s *CategoryRepositoryTestSuite) TestGetAllCategories_Success() {
	categories := []domain.Category{
		{Name: "Category 1", Description: "Description for category 1"},
		{Name: "Category 2", Description: "Description for category 2"},
	}

	for i := range categories {
		categories[i].BindID()
		err := s.repo.Create(&categories[i])
		assert.Nil(s.T(), err)
	}

	retrieved, err := s.repo.GetAll()

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), len(categories), len(retrieved))
}

func (s *CategoryRepositoryTestSuite) TestGetAllCategories_Empty() {
	retrieved, err := s.repo.GetAll()

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, len(retrieved))
}

func (s *CategoryRepositoryTestSuite) TestGetCategoryByID_Success() {
	category := &domain.Category{
		Name:        "Test Category",
		Description: "This is a test category description.",
	}
	category.BindID()
	err := s.repo.Create(category)
	assert.Nil(s.T(), err)

	retrieved, err := s.repo.GetByID(category.ID)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), category.ID, retrieved.ID)
}

func (s *CategoryRepositoryTestSuite) TestGetCategoryByID_NotFound() {
	id := uuid.New()
	category, err := s.repo.GetByID(id)

	assert.Nil(s.T(), category)
	assert.Error(s.T(), err)
	assert.ErrorAs(s.T(), err, &gorm.ErrRecordNotFound)
}

func (s *CategoryRepositoryTestSuite) TestUpdateCategory_Success() {
	category := &domain.Category{
		Name:        "Test Category",
		Description: "This is a test category description.",
	}
	category.BindID()
	err := s.repo.Create(category)
	assert.Nil(s.T(), err)

	category.Name = "Updated Category"
	err = s.repo.Update(category)
	assert.Nil(s.T(), err)

	updated, err := s.repo.GetByID(category.ID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "Updated Category", updated.Name)
}

func (s *CategoryRepositoryTestSuite) TestDeleteCategory_Success() {
	category := &domain.Category{
		Name:        "Test Category",
		Description: "This is a test category description.",
	}
	category.BindID()
	err := s.repo.Create(category)
	assert.Nil(s.T(), err)

	err = s.repo.Delete(category.ID)
	assert.Nil(s.T(), err)

	deleted, err := s.repo.GetByID(category.ID)
	assert.Nil(s.T(), deleted)
	assert.ErrorAs(s.T(), err, &gorm.ErrRecordNotFound)
}

func TestCategoryRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(CategoryRepositoryTestSuite))
}
