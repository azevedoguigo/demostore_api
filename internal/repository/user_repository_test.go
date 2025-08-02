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

type UserRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo *repository.UserRepository
}

func (s *UserRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	db.AutoMigrate(&domain.User{})
	s.repo = repository.NewUserRepository(db)
}

func (s *UserRepositoryTestSuite) TestCreateUser() {
	user := &domain.User{
		Name:     "Test User",
		Email:    "testuser@example.com",
		Password: "password",
	}
	err := s.repo.Create(user)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), user.ID, "User ID should be set after creation")
}

func (s *UserRepositoryTestSuite) TestGetUserByID_Success() {
	user := &domain.User{
		Name:     "Test User",
		Email:    "testuser@example.com",
		Password: "password",
	}
	err := s.repo.Create(user)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), user.ID, "User ID should be set during creation")
	assert.Equal(s.T(), user.Name, "Test User")
	assert.Equal(s.T(), user.Email, "testuser@example.com")
}

func (s *UserRepositoryTestSuite) TestGetUserByID_NotFound() {
	id := "123e4567-e89b-12d3-a456-426614174000"
	user, err := s.repo.GetByID(uuid.MustParse(id))

	assert.Nil(s.T(), user, "User should be nil when not found")
	assert.Error(s.T(), err, "Expected an error when user is not found")
	assert.ErrorAs(s.T(), err, &gorm.ErrRecordNotFound, "Expected gorm.ErrRecordNotFound error")
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
