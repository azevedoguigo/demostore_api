package service_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

type UserServiceTestSuite struct {
	suite.Suite
	repo    *MockUserRepository
	service *service.UserServiceImpl
}

func (suite *UserServiceTestSuite) SetupTest() {
	suite.repo = new(MockUserRepository)
	suite.service = service.NewUserService(suite.repo)
}

func (suite *UserServiceTestSuite) TestCreateUser_Success() {
	user := &domain.User{Name: "Test User", Email: "test@example.com"}

	suite.repo.On("Create", user).Return(nil).Once()

	err := suite.service.CreateUser(user)

	assert.NoError(suite.T(), err)
	suite.repo.AssertExpectations(suite.T())
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
