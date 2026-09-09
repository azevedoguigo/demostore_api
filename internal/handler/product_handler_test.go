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

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) CreateProduct(dto request.CreateProductRequestDTO) error {
	args := m.Called(dto)
	return args.Error(0)
}

func (m *MockProductService) GetAllProducts() ([]domain.Product, error) {
	args := m.Called()
	return args.Get(0).([]domain.Product), args.Error(1)
}

func (m *MockProductService) GetProductByID(id string) (*domain.Product, error) {
	args := m.Called(id)
	p := args.Get(0)
	if p == nil {
		return nil, args.Error(1)
	}
	return p.(*domain.Product), args.Error(1)
}

func (m *MockProductService) UpdateProduct(id string, dto request.UpdateProductRequestDTO) error {
	args := m.Called(id, dto)
	return args.Error(0)
}

func (m *MockProductService) DeleteProduct(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

type ProductHandlerTestSuite struct {
	suite.Suite
	handler *handler.ProductHandler
	service *MockProductService
}

func (suite *ProductHandlerTestSuite) SetupTest() {
	suite.service = new(MockProductService)
	suite.handler = handler.NewProductHandler(suite.service)
}

func (suite *ProductHandlerTestSuite) TestCreateProduct_Success() {
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "A product for testing",
		Price:       99.99,
	}
	body, _ := json.Marshal(dto)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("CreateProduct", dto).Return(nil)

	suite.handler.CreateProduct(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusCreated, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Product created successfully", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestCreateProduct_InvalidBody() {
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte("invalid")))
	recorder := httptest.NewRecorder()

	suite.handler.CreateProduct(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Bad Request", respBody["error"])
	suite.Equal("Invalid request body", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestCreateProduct_InternalServerError() {
	dto := request.CreateProductRequestDTO{
		Name:        "Test Product",
		Description: "A product for testing",
		Price:       99.99,
	}
	body, _ := json.Marshal(dto)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	suite.service.On("CreateProduct", dto).Return(assert.AnError)

	suite.handler.CreateProduct(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestGetAllProducts_Success() {
	products := []domain.Product{
		{
			Name:        "Product 1",
			Description: "Description for product 1",
			Price:       10.0,
			Stock:       5,
		},
		{
			Name:        "Product 2",
			Description: "Description for product 2",
			Price:       20.0,
			Stock:       15,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	recorder := httptest.NewRecorder()

	suite.service.On("GetAllProducts").Return(products, nil)

	suite.handler.GetAllProducts(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody []domain.Product
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(products, respBody)
}

func (suite *ProductHandlerTestSuite) TestGetAllProducts_InternalServerError() {
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	recorder := httptest.NewRecorder()
	suite.service.On("GetAllProducts").Return([]domain.Product(nil), assert.AnError)

	suite.handler.GetAllProducts(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *ProductHandlerTestSuite) serveWithChiParam(method, pattern, path string, h http.HandlerFunc) *http.Response {
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

func (suite *ProductHandlerTestSuite) TestGetProductByID_Success() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	product := &domain.Product{
		Name:        "Product 1",
		Description: "Description for product 1",
		Price:       10.0,
		Stock:       5,
	}

	suite.service.On("GetProductByID", productID).Return(product, nil)

	resp := suite.serveWithChiParam(http.MethodGet, "/products/{id}", "/products/"+productID, suite.handler.GetProductByID)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody domain.Product
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(*product, respBody)
}

func (suite *ProductHandlerTestSuite) TestGetProductByID_InvalidID() {
	resp := suite.serveWithChiParam(http.MethodGet, "/products/{id}", "/products/not-a-uuid", suite.handler.GetProductByID)
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Bad Request", respBody["error"])
}

func (suite *ProductHandlerTestSuite) TestGetProductByID_InternalServerError() {
	productID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("GetProductByID", productID).Return((*domain.Product)(nil), assert.AnError)

	resp := suite.serveWithChiParam(http.MethodGet, "/products/{id}", "/products/"+productID, suite.handler.GetProductByID)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestGetProductByID_NotFound() {
	productID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("GetProductByID", productID).Return((*domain.Product)(nil), nil)

	resp := suite.serveWithChiParam(http.MethodGet, "/products/{id}", "/products/"+productID, suite.handler.GetProductByID)
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Not Found", respBody["error"])
	suite.Equal("Product not found", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestUpdateProduct_Success() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	dto := request.UpdateProductRequestDTO{
		Name:        "Updated Product",
		Description: "Updated description",
		Price:       50.0,
		Stock:       10,
	}
	body, _ := json.Marshal(dto)

	suite.service.On("UpdateProduct", productID, dto).Return(nil)

	r := chi.NewRouter()
	r.Put("/products/{id}", suite.handler.UpdateProduct)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID, bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Product updated successfully", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestUpdateProduct_InvalidID() {
	dto := request.UpdateProductRequestDTO{
		Name:        "Updated Product",
		Description: "Updated description",
		Price:       50.0,
		Stock:       10,
	}
	body, _ := json.Marshal(dto)

	r := chi.NewRouter()
	r.Put("/products/{id}", suite.handler.UpdateProduct)
	req := httptest.NewRequest(http.MethodPut, "/products/not-a-uuid", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	suite.Equal(http.StatusBadRequest, recorder.Result().StatusCode)
}

func (suite *ProductHandlerTestSuite) TestUpdateProduct_InvalidBody() {
	productID := "123e4567-e89b-12d3-a456-426614174000"

	r := chi.NewRouter()
	r.Put("/products/{id}", suite.handler.UpdateProduct)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID, bytes.NewReader([]byte("invalid")))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Invalid request body", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestUpdateProduct_NotFound() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	dto := request.UpdateProductRequestDTO{
		Name:        "Updated Product",
		Description: "Updated description",
		Price:       50.0,
		Stock:       10,
	}
	body, _ := json.Marshal(dto)

	suite.service.On("UpdateProduct", productID, dto).Return(gorm.ErrRecordNotFound)

	r := chi.NewRouter()
	r.Put("/products/{id}", suite.handler.UpdateProduct)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID, bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Product not found", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestUpdateProduct_InternalServerError() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	dto := request.UpdateProductRequestDTO{
		Name:        "Updated Product",
		Description: "Updated description",
		Price:       50.0,
		Stock:       10,
	}
	body, _ := json.Marshal(dto)

	suite.service.On("UpdateProduct", productID, dto).Return(assert.AnError)

	r := chi.NewRouter()
	r.Put("/products/{id}", suite.handler.UpdateProduct)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID, bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestDeleteProduct_Success() {
	productID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("DeleteProduct", productID).Return(nil)

	resp := suite.serveWithChiParam(http.MethodDelete, "/products/{id}", "/products/"+productID, suite.handler.DeleteProduct)
	suite.Equal(http.StatusNoContent, resp.StatusCode)
}

func (suite *ProductHandlerTestSuite) TestDeleteProduct_InvalidID() {
	resp := suite.serveWithChiParam(http.MethodDelete, "/products/{id}", "/products/not-a-uuid", suite.handler.DeleteProduct)
	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *ProductHandlerTestSuite) TestDeleteProduct_NotFound() {
	productID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("DeleteProduct", productID).Return(gorm.ErrRecordNotFound)

	resp := suite.serveWithChiParam(http.MethodDelete, "/products/{id}", "/products/"+productID, suite.handler.DeleteProduct)
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Product not found", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestDeleteProduct_InternalServerError() {
	productID := "123e4567-e89b-12d3-a456-426614174000"

	suite.service.On("DeleteProduct", productID).Return(assert.AnError)

	resp := suite.serveWithChiParam(http.MethodDelete, "/products/{id}", "/products/"+productID, suite.handler.DeleteProduct)
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func TestProductHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ProductHandlerTestSuite))
}
