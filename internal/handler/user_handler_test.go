package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
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

func (suite *UserHandlerTestSuite) TestCreateUser_InvalidRequestBody() {
	req := httptest.NewRequest("POST", "/users", bytes.NewReader([]byte("invalid json")))
	rr := httptest.NewRecorder()

	suite.handler.CreateUser(rr, req)

	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	assert.Equal(suite.T(), "Bad Request", response["error"])
	assert.Equal(suite.T(), "Invalid request body", response["message"])
}

func (suite *UserHandlerTestSuite) TestCreateUser_InternalServerError() {
	user := &domain.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "passwd123",
	}
	suite.service.On("CreateUser", user).Return(errors.New("internal server error")).Once()

	body, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	suite.handler.CreateUser(rr, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, rr.Code)

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	assert.Equal(suite.T(), "Internal Server Error", response["error"])
	assert.Equal(suite.T(), "internal server error", response["message"])
}

func (suite *UserHandlerTestSuite) TestGetUserByID_Success() {
	userID := uuid.New()
	user := &domain.User{
		ID:    userID,
		Name:  "Test User",
		Email: "test@example.com",
	}

	suite.service.On("GetUserByID", userID).Return(user, nil).Once()

	r := chi.NewRouter()
	r.Get("/users/{id}", suite.handler.GetUserByID)
	req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	assert.Equal(suite.T(), "application/json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	assert.Equal(suite.T(), user.ID.String(), response["id"])
	assert.Equal(suite.T(), user.Name, response["name"])
	assert.Equal(suite.T(), user.Email, response["email"])
}

func (suite *UserHandlerTestSuite) TestGetUserByID_NotFound() {
	userID := uuid.New()
	suite.service.On("GetUserByID", userID).Return(nil, gorm.ErrRecordNotFound).Once()

	r := chi.NewRouter()
	r.Get("/users/{id}", suite.handler.GetUserByID)
	req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusNotFound, rr.Code)

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	assert.Equal(suite.T(), "User not found", response["message"])
}

func (suite *UserHandlerTestSuite) TestGetUserByID_InvalidUserID() {
	r := chi.NewRouter()
	r.Get("/users/{id}", suite.handler.GetUserByID)
	req := httptest.NewRequest("GET", "/users/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	assert.Equal(suite.T(), "Bad Request", response["error"])
	assert.Equal(suite.T(), "invalid UUID length: 12", response["message"])
}

func (suite *UserHandlerTestSuite) TestGetUserByID_InternalServerError() {
	userID := uuid.New()
	suite.service.On("GetUserByID", userID).Return(nil, errors.New("internal server error")).Once()

	r := chi.NewRouter()
	r.Get("/users/{id}", suite.handler.GetUserByID)
	req := httptest.NewRequest("GET", "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, rr.Code)

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	assert.Equal(suite.T(), "Internal Server Error", response["error"])
	assert.Equal(suite.T(), "internal server error", response["message"])
}

func TestUserHandlerSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}
