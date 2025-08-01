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

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
