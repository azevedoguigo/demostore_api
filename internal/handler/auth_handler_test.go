package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(email, password string) (string, error) {
	args := m.Called(email, password)
	return args.String(0), args.Error(1)
}

type AuthHandlerTestSuite struct {
	suite.Suite
	handler *handler.AuthHandler
	service *MockAuthService
}

func (suite *AuthHandlerTestSuite) SetupTest() {
	suite.service = new(MockAuthService)
	suite.handler = handler.NewAuthHandler(suite.service)
}

func (suite *AuthHandlerTestSuite) TestLogin_Success() {
	loginReq := request.LoginRequestDTO{
		Email:    "test@example.com",
		Password: "password",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("Login", loginReq.Email, loginReq.Password).Return("token123", nil)

	suite.handler.Login(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusOK, resp.StatusCode)
	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)
	suite.Equal("token123", respBody["access_token"])
}

func (suite *AuthHandlerTestSuite) TestLogin_InvalidBody() {
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("invalid")))
	recorder := httptest.NewRecorder()

	suite.handler.Login(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *AuthHandlerTestSuite) TestLogin_InvalidPassword() {
	loginReq := request.LoginRequestDTO{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("Login", loginReq.Email, loginReq.Password).Return("", errors.New("invalid password"))

	suite.handler.Login(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *AuthHandlerTestSuite) TestLogin_UserNotFound() {
	loginReq := request.LoginRequestDTO{
		Email:    "notfound@example.com",
		Password: "password",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("Login", loginReq.Email, loginReq.Password).Return("", gorm.ErrRecordNotFound)

	suite.handler.Login(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusNotFound, resp.StatusCode)
}

func (suite *AuthHandlerTestSuite) TestLogin_InternalError() {
	loginReq := request.LoginRequestDTO{
		Email:    "test@example.com",
		Password: "password",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("Login", loginReq.Email, loginReq.Password).Return("", errors.New("unexpected error"))

	suite.handler.Login(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}
