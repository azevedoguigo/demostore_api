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

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	if user, ok := args.Get(0).(*domain.User); ok {
		return user, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if user, ok := args.Get(0).(*domain.User); ok {
		return user, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockUserRepository) Update(user *domain.User) error {
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
	dto := &request.CreateUserRequestDTO{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "passwd123",
	}

	suite.repo.On("Create", mock.MatchedBy(func(user *domain.User) bool {
		return user.Role == domain.RoleCustomer
	})).Return(nil).Once()

	err := suite.service.CreateUser(dto)

	assert.NoError(suite.T(), err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestCreateUser_Error() {
	dto := &request.CreateUserRequestDTO{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "passwd123",
	}

	suite.repo.On("Create", mock.Anything).Return(assert.AnError).Once()

	err := suite.service.CreateUser(dto)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), assert.AnError, err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserByID_Success() {
	id := uuid.New()
	user := &domain.User{
		ID:    id,
		Name:  "Test User",
		Email: "test@example.com",
	}

	suite.repo.On("GetByID", id).Return(user, nil).Once()

	result, err := suite.service.GetUserByID(id)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user, result)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserByID_NotFound() {
	id := uuid.New()

	suite.repo.On("GetByID", id).Return(nil, gorm.ErrRecordNotFound).Once()

	result, err := suite.service.GetUserByID(id)

	assert.Nil(suite.T(), result)
	assert.ErrorAs(suite.T(), err, &gorm.ErrRecordNotFound)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserByID_Error() {
	id := uuid.New()

	suite.repo.On("GetByID", id).Return(nil, assert.AnError).Once()

	result, err := suite.service.GetUserByID(id)

	assert.Nil(suite.T(), result)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), assert.AnError, err)
	suite.repo.AssertExpectations(suite.T())
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
