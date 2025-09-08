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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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
	return args.Get(0).(*domain.Product), args.Error(1)
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

func (suite *ProductHandlerTestSuite) TestGetProductByID_Success() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	product := &domain.Product{
		Name:        "Product 1",
		Description: "Description for product 1",
		Price:       10.0,
		Stock:       5,
	}

	req := httptest.NewRequest(http.MethodGet, "/products?id="+productID, nil)
	recorder := httptest.NewRecorder()

	suite.service.On("GetProductByID", productID).Return(product, nil)

	suite.handler.GetProductByID(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusOK, resp.StatusCode)

	var respBody domain.Product
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal(*product, respBody)
}

func (suite *ProductHandlerTestSuite) TestGetProductByID_MissingID() {
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	recorder := httptest.NewRecorder()

	suite.handler.GetProductByID(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Bad Request", respBody["error"])
	suite.Equal("Missing product ID", respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestGetProductByID_InternalServerError() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	req := httptest.NewRequest(http.MethodGet, "/products?id="+productID, nil)
	recorder := httptest.NewRecorder()

	suite.service.On("GetProductByID", productID).Return((*domain.Product)(nil), assert.AnError)

	suite.handler.GetProductByID(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Internal Server Error", respBody["error"])
	suite.Equal(assert.AnError.Error(), respBody["message"])
}

func (suite *ProductHandlerTestSuite) TestGetProductByID_NotFound() {
	productID := "123e4567-e89b-12d3-a456-426614174000"
	req := httptest.NewRequest(http.MethodGet, "/products?id="+productID, nil)
	recorder := httptest.NewRecorder()

	suite.service.On("GetProductByID", productID).Return((*domain.Product)(nil), nil)

	suite.handler.GetProductByID(recorder, req)
	resp := recorder.Result()
	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	suite.Equal("Not Found", respBody["error"])
	suite.Equal("Product not found", respBody["message"])
}

func TestProductHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ProductHandlerTestSuite))
}
