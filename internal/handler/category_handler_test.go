package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type MockCategoryService struct {
	mock.Mock
}

func (m *MockCategoryService) CreateCategory(dto request.CreateCategoryRequestDTO) error {
	args := m.Called(dto)
	return args.Error(0)
}

func (m *MockCategoryService) GetAllCategories() ([]domain.Category, error) {
	args := m.Called()
	return args.Get(0).([]domain.Category), args.Error(1)
}

func (m *MockCategoryService) GetCategoryByID(id string) (*domain.Category, error) {
	args := m.Called(id)
	c := args.Get(0)
	if c == nil {
		return nil, args.Error(1)
	}
	return c.(*domain.Category), args.Error(1)
}

func (m *MockCategoryService) UpdateCategory(id string, dto request.UpdateCategoryRequestDTO) error {
	args := m.Called(id, dto)
	return args.Error(0)
}

func (m *MockCategoryService) DeleteCategory(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

type CategoryHandlerTestSuite struct {
	suite.Suite
	handler *handler.CategoryHandler
	service *MockCategoryService
}

func (suite *CategoryHandlerTestSuite) SetupTest() {
	suite.service = new(MockCategoryService)
	suite.handler = handler.NewCategoryHandler(suite.service)
}

func (suite *CategoryHandlerTestSuite) serveWithChiParam(method, pattern, path string, h http.HandlerFunc) *http.Response {
	r := chi.NewRouter()
	switch method {
	case http.MethodGet:
		r.Get(pattern, h)
	case http.MethodPut:
		r.Put(pattern, h)
	case http.MethodDelete:
		r.Delete(pattern, h)
	}

	req := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	return recorder.Result()
}

func (suite *CategoryHandlerTestSuite) TestCreateCategory_Success() {
	dto := request.CreateCategoryRequestDTO{
		Name:        "Test Category",
		Description: "A category for testing",
	}
	body, _ := json.Marshal(dto)
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("CreateCategory", dto).Return(nil)

	suite.handler.CreateCategory(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusCreated, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Category created successfully", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestCreateCategory_InvalidBody() {
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader([]byte("invalid")))
	recorder := httptest.NewRecorder()

	suite.handler.CreateCategory(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Bad Request", respBody["error"])
	suite.Equal("Invalid request body", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestCreateCategory_InternalServerError() {
	dto := request.CreateCategoryRequestDTO{
		Name:        "Test Category",
		Description: "A category for testing",
	}
	body, _ := json.Marshal(dto)
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("CreateCategory", dto).Return(assert.AnError)

	suite.handler.CreateCategory(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestGetAllCategories_Success() {
	categories := []domain.Category{
		{Name: "Category 1", Description: "Description for category 1"},
		{Name: "Category 2", Description: "Description for category 2"},
	}

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	recorder := httptest.NewRecorder()

	suite.service.On("GetAllCategories").Return(categories, nil)

	suite.handler.GetAllCategories(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody []domain.Category
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(categories, respBody)
}

func (suite *CategoryHandlerTestSuite) TestGetAllCategories_InternalServerError() {
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	recorder := httptest.NewRecorder()
	suite.service.On("GetAllCategories").Return([]domain.Category(nil), assert.AnError)

	suite.handler.GetAllCategories(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestGetCategoryByID_Success() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"
	category := &domain.Category{
		Name:        "Category 1",
		Description: "Description for category 1",
	}

	suite.service.On("GetCategoryByID", categoryID).Return(category, nil)

	resp := suite.serveWithChiParam(http.MethodGet, "/categories/{id}", "/categories/"+categoryID, suite.handler.GetCategoryByID)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody domain.Category
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(*category, respBody)
}

func (suite *CategoryHandlerTestSuite) TestGetCategoryByID_InvalidID() {
	resp := suite.serveWithChiParam(http.MethodGet, "/categories/{id}", "/categories/not-a-uuid", suite.handler.GetCategoryByID)
	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *CategoryHandlerTestSuite) TestGetCategoryByID_InternalServerError() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("GetCategoryByID", categoryID).Return((*domain.Category)(nil), assert.AnError)

	resp := suite.serveWithChiParam(http.MethodGet, "/categories/{id}", "/categories/"+categoryID, suite.handler.GetCategoryByID)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestGetCategoryByID_NotFound() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("GetCategoryByID", categoryID).Return((*domain.Category)(nil), nil)

	resp := suite.serveWithChiParam(http.MethodGet, "/categories/{id}", "/categories/"+categoryID, suite.handler.GetCategoryByID)
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Not Found", respBody["error"])
	suite.Equal("Category not found", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestUpdateCategory_Success() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"
	dto := request.UpdateCategoryRequestDTO{
		Name:        "Updated Category",
		Description: "Updated description",
	}
	body, _ := json.Marshal(dto)

	suite.service.On("UpdateCategory", categoryID, dto).Return(nil)

	r := chi.NewRouter()
	r.Put("/categories/{id}", suite.handler.UpdateCategory)
	req := httptest.NewRequest(http.MethodPut, "/categories/"+categoryID, bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Category updated successfully", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestUpdateCategory_InvalidID() {
	dto := request.UpdateCategoryRequestDTO{
		Name:        "Updated Category",
		Description: "Updated description",
	}
	body, _ := json.Marshal(dto)

	r := chi.NewRouter()
	r.Put("/categories/{id}", suite.handler.UpdateCategory)
	req := httptest.NewRequest(http.MethodPut, "/categories/not-a-uuid", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	suite.Equal(http.StatusBadRequest, recorder.Result().StatusCode)
}

func (suite *CategoryHandlerTestSuite) TestUpdateCategory_InvalidBody() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"

	r := chi.NewRouter()
	r.Put("/categories/{id}", suite.handler.UpdateCategory)
	req := httptest.NewRequest(http.MethodPut, "/categories/"+categoryID, bytes.NewReader([]byte("invalid")))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Invalid request body", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestUpdateCategory_NotFound() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"
	dto := request.UpdateCategoryRequestDTO{
		Name:        "Updated Category",
		Description: "Updated description",
	}
	body, _ := json.Marshal(dto)

	suite.service.On("UpdateCategory", categoryID, dto).Return(gorm.ErrRecordNotFound)

	r := chi.NewRouter()
	r.Put("/categories/{id}", suite.handler.UpdateCategory)
	req := httptest.NewRequest(http.MethodPut, "/categories/"+categoryID, bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Category not found", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestUpdateCategory_InternalServerError() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"
	dto := request.UpdateCategoryRequestDTO{
		Name:        "Updated Category",
		Description: "Updated description",
	}
	body, _ := json.Marshal(dto)

	suite.service.On("UpdateCategory", categoryID, dto).Return(assert.AnError)

	r := chi.NewRouter()
	r.Put("/categories/{id}", suite.handler.UpdateCategory)
	req := httptest.NewRequest(http.MethodPut, "/categories/"+categoryID, bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestDeleteCategory_Success() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("DeleteCategory", categoryID).Return(nil)

	resp := suite.serveWithChiParam(http.MethodDelete, "/categories/{id}", "/categories/"+categoryID, suite.handler.DeleteCategory)
	suite.Equal(http.StatusNoContent, resp.StatusCode)
}

func (suite *CategoryHandlerTestSuite) TestDeleteCategory_InvalidID() {
	resp := suite.serveWithChiParam(http.MethodDelete, "/categories/{id}", "/categories/not-a-uuid", suite.handler.DeleteCategory)
	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *CategoryHandlerTestSuite) TestDeleteCategory_NotFound() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("DeleteCategory", categoryID).Return(gorm.ErrRecordNotFound)

	resp := suite.serveWithChiParam(http.MethodDelete, "/categories/{id}", "/categories/"+categoryID, suite.handler.DeleteCategory)
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Category not found", respBody["message"])
}

func (suite *CategoryHandlerTestSuite) TestDeleteCategory_InternalServerError() {
	categoryID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("DeleteCategory", categoryID).Return(assert.AnError)

	resp := suite.serveWithChiParam(http.MethodDelete, "/categories/{id}", "/categories/"+categoryID, suite.handler.DeleteCategory)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func TestCategoryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CategoryHandlerTestSuite))
}
