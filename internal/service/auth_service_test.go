package service_test

import (
	"errors"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) GetUserByID(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	u := args.Get(0)
	if u == nil {
		return nil, args.Error(1)
	}
	return u.(*domain.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	u := args.Get(0)
	if u == nil {
		return nil, args.Error(1)
	}
	return u.(*domain.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(dto *request.UpdateUserRequestDTO) error {
	args := m.Called(dto)
	return args.Error(0)
}

type AuthServiceTestSuite struct {
	suite.Suite
	mockUserService *MockUserService
	authService     *service.AuthServiceImpl
}

func (suite *AuthServiceTestSuite) SetupTest() {
	suite.mockUserService = new(MockUserService)
	suite.authService = service.NewAuthService(suite.mockUserService)
}

func (suite *AuthServiceTestSuite) TestLogin_Success() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	suite.Require().NoError(err)
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		Password:    string(hashedPassword),
		AccessToken: "",
	}
	suite.mockUserService.On("GetUserByEmail", "test@example.com").Return(user, nil)
	suite.mockUserService.On("UpdateUser", mock.Anything).Return(nil)

	token, err := suite.authService.Login("test@example.com", "password")
	suite.NoError(err)
	suite.NotEmpty(token)
	suite.mockUserService.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_UserNotFound() {
	suite.mockUserService.On("GetUserByEmail", "notfound@example.com").Return(nil, errors.New("record not found"))

	token, err := suite.authService.Login("notfound@example.com", "password")
	suite.Error(err)
	suite.Empty(token)
	suite.mockUserService.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidPassword() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	suite.Require().NoError(err)
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		Password:    string(hashedPassword),
		AccessToken: "",
	}
	suite.mockUserService.On("GetUserByEmail", "test@example.com").Return(user, nil)

	token, err := suite.authService.Login("test@example.com", "wrongpassword")
	suite.Error(err)
	suite.Equal("invalid password", err.Error())
	suite.Empty(token)
	suite.mockUserService.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_UpdateUserError() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	suite.Require().NoError(err)
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		Password:    string(hashedPassword),
		AccessToken: "",
	}
	suite.mockUserService.On("GetUserByEmail", "test@example.com").Return(user, nil)
	suite.mockUserService.On("UpdateUser", mock.Anything).Return(errors.New("update error"))

	token, err := suite.authService.Login("test@example.com", "password")
	suite.Error(err)
	suite.Equal("update error", err.Error())
	suite.Empty(token)
	suite.mockUserService.AssertExpectations(suite.T())
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
