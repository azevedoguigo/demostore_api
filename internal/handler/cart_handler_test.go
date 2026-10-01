package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/response"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockCartService struct {
	mock.Mock
}

func (m *MockCartService) cartResult(args mock.Arguments) (*domain.Cart, error) {
	c := args.Get(0)
	if c == nil {
		return nil, args.Error(1)
	}
	return c.(*domain.Cart), args.Error(1)
}

func (m *MockCartService) GetCart(userID uuid.UUID) (*domain.Cart, error) {
	return m.cartResult(m.Called(userID))
}

func (m *MockCartService) AddItem(userID uuid.UUID, dto request.AddCartItemRequestDTO) (*domain.Cart, error) {
	return m.cartResult(m.Called(userID, dto))
}

func (m *MockCartService) UpdateItem(userID uuid.UUID, productID string, dto request.UpdateCartItemRequestDTO) (*domain.Cart, error) {
	return m.cartResult(m.Called(userID, productID, dto))
}

func (m *MockCartService) RemoveItem(userID uuid.UUID, productID string) (*domain.Cart, error) {
	return m.cartResult(m.Called(userID, productID))
}

func (m *MockCartService) ClearCart(userID uuid.UUID) error {
	return m.Called(userID).Error(0)
}

type CartHandlerTestSuite struct {
	suite.Suite
	handler *handler.CartHandler
	service *MockCartService
	userID  uuid.UUID
	cart    *domain.Cart
	product *domain.Product
	router  *chi.Mux
}

func (suite *CartHandlerTestSuite) SetupTest() {
	suite.service = new(MockCartService)
	suite.handler = handler.NewCartHandler(suite.service)
	suite.userID = uuid.New()
	suite.product = &domain.Product{ID: uuid.New(), Name: "Product", Price: 10, Stock: 5}
	suite.cart = &domain.Cart{
		ID:     uuid.New(),
		UserID: suite.userID,
		Items:  []domain.CartItem{{ID: uuid.New(), ProductID: suite.product.ID, Product: suite.product, Quantity: 3}},
	}

	suite.router = chi.NewRouter()
	suite.router.Get("/cart", suite.handler.GetCart)
	suite.router.Delete("/cart", suite.handler.ClearCart)
	suite.router.Post("/cart/items", suite.handler.AddItem)
	suite.router.Put("/cart/items/{product_id}", suite.handler.UpdateItem)
	suite.router.Delete("/cart/items/{product_id}", suite.handler.RemoveItem)
}

func (suite *CartHandlerTestSuite) do(method, path string, body any, authenticated bool) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		buf.WriteString(b)
	default:
		json.NewEncoder(&buf).Encode(b)
	}

	req := httptest.NewRequest(method, path, &buf)
	if authenticated {
		token := jwtauth.New("HS256", []byte("secret"), nil)
		jwt, _, _ := token.Encode(map[string]interface{}{"jti": suite.userID.String()})
		req = req.WithContext(jwtauth.NewContext(context.Background(), jwt, nil))
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *CartHandlerTestSuite) TestGetCart_Success() {
	suite.service.On("GetCart", suite.userID).Return(suite.cart, nil)

	w := suite.do(http.MethodGet, "/cart", nil, true)

	suite.Equal(http.StatusOK, w.Code)
	var resp response.CartResponse
	suite.NoError(json.NewDecoder(w.Body).Decode(&resp))
	suite.Equal(suite.cart.ID, resp.ID)
	suite.Len(resp.Items, 1)
	suite.Equal(30.0, resp.Items[0].Subtotal)
	suite.Equal(30.0, resp.Total)
}

func (suite *CartHandlerTestSuite) TestGetCart_EmptyCartReturnsEmptyArray() {
	suite.service.On("GetCart", suite.userID).Return(&domain.Cart{ID: uuid.New()}, nil)

	w := suite.do(http.MethodGet, "/cart", nil, true)

	suite.Equal(http.StatusOK, w.Code)
	suite.Contains(w.Body.String(), `"items":[]`)
}

func (suite *CartHandlerTestSuite) TestGetCart_Unauthenticated() {
	w := suite.do(http.MethodGet, "/cart", nil, false)

	suite.Equal(http.StatusUnauthorized, w.Code)
	suite.service.AssertNotCalled(suite.T(), "GetCart", mock.Anything)
}

func (suite *CartHandlerTestSuite) TestGetCart_InternalError() {
	suite.service.On("GetCart", suite.userID).Return(nil, assert.AnError)

	w := suite.do(http.MethodGet, "/cart", nil, true)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *CartHandlerTestSuite) TestAddItem_Success() {
	dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: 2}
	suite.service.On("AddItem", suite.userID, dto).Return(suite.cart, nil)

	w := suite.do(http.MethodPost, "/cart/items", dto, true)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *CartHandlerTestSuite) TestAddItem_InvalidBody() {
	w := suite.do(http.MethodPost, "/cart/items", "{invalid", true)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *CartHandlerTestSuite) TestAddItem_InvalidProductID() {
	w := suite.do(http.MethodPost, "/cart/items", request.AddCartItemRequestDTO{ProductID: "x", Quantity: 1}, true)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *CartHandlerTestSuite) TestAddItem_ErrorMapping() {
	cases := map[string]struct {
		err    error
		status int
	}{
		"product not found":  {service.ErrProductNotFound, http.StatusNotFound},
		"invalid quantity":   {service.ErrInvalidQuantity, http.StatusBadRequest},
		"insufficient stock": {service.ErrInsufficientStock, http.StatusConflict},
		"unexpected":         {assert.AnError, http.StatusInternalServerError},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			svc := new(MockCartService)
			suite.handler = handler.NewCartHandler(svc)
			suite.router = chi.NewRouter()
			suite.router.Post("/cart/items", suite.handler.AddItem)

			dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: 1}
			svc.On("AddItem", suite.userID, dto).Return(nil, tc.err)

			w := suite.do(http.MethodPost, "/cart/items", dto, true)

			suite.Equal(tc.status, w.Code)
		})
	}
}

func (suite *CartHandlerTestSuite) TestUpdateItem_Success() {
	dto := request.UpdateCartItemRequestDTO{Quantity: 4}
	suite.service.On("UpdateItem", suite.userID, suite.product.ID.String(), dto).Return(suite.cart, nil)

	w := suite.do(http.MethodPut, "/cart/items/"+suite.product.ID.String(), dto, true)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *CartHandlerTestSuite) TestUpdateItem_InvalidProductID() {
	w := suite.do(http.MethodPut, "/cart/items/invalid", request.UpdateCartItemRequestDTO{Quantity: 1}, true)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *CartHandlerTestSuite) TestUpdateItem_InvalidBody() {
	w := suite.do(http.MethodPut, "/cart/items/"+suite.product.ID.String(), "{invalid", true)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *CartHandlerTestSuite) TestUpdateItem_ItemNotFound() {
	dto := request.UpdateCartItemRequestDTO{Quantity: 1}
	suite.service.On("UpdateItem", suite.userID, suite.product.ID.String(), dto).Return(nil, service.ErrCartItemNotFound)

	w := suite.do(http.MethodPut, "/cart/items/"+suite.product.ID.String(), dto, true)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *CartHandlerTestSuite) TestRemoveItem_Success() {
	suite.service.On("RemoveItem", suite.userID, suite.product.ID.String()).Return(suite.cart, nil)

	w := suite.do(http.MethodDelete, "/cart/items/"+suite.product.ID.String(), nil, true)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *CartHandlerTestSuite) TestRemoveItem_InvalidProductID() {
	w := suite.do(http.MethodDelete, "/cart/items/invalid", nil, true)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *CartHandlerTestSuite) TestRemoveItem_ItemNotFound() {
	suite.service.On("RemoveItem", suite.userID, suite.product.ID.String()).Return(nil, service.ErrCartItemNotFound)

	w := suite.do(http.MethodDelete, "/cart/items/"+suite.product.ID.String(), nil, true)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *CartHandlerTestSuite) TestClearCart_Success() {
	suite.service.On("ClearCart", suite.userID).Return(nil)

	w := suite.do(http.MethodDelete, "/cart", nil, true)

	suite.Equal(http.StatusNoContent, w.Code)
}

func (suite *CartHandlerTestSuite) TestClearCart_InternalError() {
	suite.service.On("ClearCart", suite.userID).Return(assert.AnError)

	w := suite.do(http.MethodDelete, "/cart", nil, true)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func TestCartHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CartHandlerTestSuite))
}
