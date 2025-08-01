package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

var _ service.UserService = (*MockUserService)(nil)

type UserHandlerTestSuite struct {
	suite.Suite
	service *MockUserService
	handler *handler.UserHandler
}

func (suite *UserHandlerTestSuite) SetupTest() {
	suite.service = new(MockUserService)
	suite.handler = handler.NewUserHandler(suite.service)
}

func (suite *UserHandlerTestSuite) TestCreateUser_Success() {
	user := &domain.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "passwd123",
	}
	suite.service.On("CreateUser", user).Return(nil).Once()

	body, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	suite.handler.CreateUser(rr, req)

	assert.Equal(suite.T(), http.StatusCreated, rr.Code)

	assert.Equal(suite.T(), user.Name, "Test User")
	assert.Equal(suite.T(), user.Email, "test@example.com")
}

func TestUserHandlerSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}
