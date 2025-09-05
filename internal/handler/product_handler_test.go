package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestProductHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ProductHandlerTestSuite))
}
