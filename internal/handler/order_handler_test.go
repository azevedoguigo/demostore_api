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
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) orderResult(args mock.Arguments) (*domain.Order, error) {
	o := args.Get(0)
	if o == nil {
		return nil, args.Error(1)
	}
	return o.(*domain.Order), args.Error(1)
}

func (m *MockOrderService) ordersResult(args mock.Arguments) ([]domain.Order, error) {
	o := args.Get(0)
	if o == nil {
		return nil, args.Error(1)
	}
	return o.([]domain.Order), args.Error(1)
}

func (m *MockOrderService) Checkout(userID uuid.UUID) (*domain.Order, error) {
	return m.orderResult(m.Called(userID))
}

func (m *MockOrderService) GetUserOrders(userID uuid.UUID) ([]domain.Order, error) {
	return m.ordersResult(m.Called(userID))
}

func (m *MockOrderService) GetOrder(userID uuid.UUID, isAdmin bool, id string) (*domain.Order, error) {
	return m.orderResult(m.Called(userID, isAdmin, id))
}

func (m *MockOrderService) CancelOrder(userID uuid.UUID, id string) (*domain.Order, error) {
	return m.orderResult(m.Called(userID, id))
}

func (m *MockOrderService) GetAllOrders() ([]domain.Order, error) {
	return m.ordersResult(m.Called())
}

func (m *MockOrderService) UpdateOrderStatus(id string, dto request.UpdateOrderStatusRequestDTO) (*domain.Order, error) {
	return m.orderResult(m.Called(id, dto))
}

type OrderHandlerTestSuite struct {
	suite.Suite
	service *MockOrderService
	router  *chi.Mux
	userID  uuid.UUID
	order   *domain.Order
}

func (suite *OrderHandlerTestSuite) SetupTest() {
	suite.service = new(MockOrderService)
	h := handler.NewOrderHandler(suite.service)
	suite.userID = uuid.New()
	suite.order = &domain.Order{ID: uuid.New(), UserID: suite.userID, Status: domain.OrderStatusPending, TotalAmount: 1999}

	suite.router = chi.NewRouter()
	suite.router.Post("/orders", h.Checkout)
	suite.router.Get("/orders", h.GetMyOrders)
	suite.router.Get("/orders/{id}", h.GetOrder)
	suite.router.Post("/orders/{id}/cancel", h.CancelOrder)
	suite.router.Get("/admin/orders", h.GetAllOrders)
	suite.router.Patch("/admin/orders/{id}/status", h.UpdateOrderStatus)
}

func (suite *OrderHandlerTestSuite) do(method, path string, body any, role string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		buf.WriteString(b)
	default:
		json.NewEncoder(&buf).Encode(b)
	}

	req := httptest.NewRequest(method, path, &buf)
	if role != "" {
		token := jwtauth.New("HS256", []byte("secret"), nil)
		jwt, _, _ := token.Encode(map[string]interface{}{"jti": suite.userID.String(), "role": role})
		req = req.WithContext(jwtauth.NewContext(context.Background(), jwt, nil))
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *OrderHandlerTestSuite) TestCheckout_Success() {
	suite.service.On("Checkout", suite.userID).Return(suite.order, nil)

	w := suite.do(http.MethodPost, "/orders", nil, domain.RoleCustomer)

	suite.Equal(http.StatusCreated, w.Code)
	var resp domain.Order
	suite.NoError(json.NewDecoder(w.Body).Decode(&resp))
	suite.Equal(suite.order.ID, resp.ID)
	suite.Equal(int64(1999), resp.TotalAmount)
}

func (suite *OrderHandlerTestSuite) TestCheckout_Unauthenticated() {
	w := suite.do(http.MethodPost, "/orders", nil, "")

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *OrderHandlerTestSuite) TestCheckout_ErrorMapping() {
	cases := map[string]struct {
		err    error
		status int
	}{
		"empty cart":         {service.ErrEmptyCart, http.StatusBadRequest},
		"product not found":  {service.ErrProductNotFound, http.StatusNotFound},
		"insufficient stock": {service.ErrInsufficientStock, http.StatusConflict},
		"unexpected":         {assert.AnError, http.StatusInternalServerError},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			suite.SetupTest()
			suite.service.On("Checkout", suite.userID).Return(nil, tc.err)

			w := suite.do(http.MethodPost, "/orders", nil, domain.RoleCustomer)

			suite.Equal(tc.status, w.Code)
		})
	}
}

func (suite *OrderHandlerTestSuite) TestGetMyOrders_EmptyReturnsArray() {
	suite.service.On("GetUserOrders", suite.userID).Return(nil, nil)

	w := suite.do(http.MethodGet, "/orders", nil, domain.RoleCustomer)

	suite.Equal(http.StatusOK, w.Code)
	suite.JSONEq(`[]`, w.Body.String())
}

func (suite *OrderHandlerTestSuite) TestGetOrder_PassesAdminFlag() {
	for role, isAdmin := range map[string]bool{domain.RoleCustomer: false, domain.RoleAdmin: true} {
		suite.Run(role, func() {
			suite.SetupTest()
			suite.service.On("GetOrder", suite.userID, isAdmin, suite.order.ID.String()).Return(suite.order, nil)

			w := suite.do(http.MethodGet, "/orders/"+suite.order.ID.String(), nil, role)

			suite.Equal(http.StatusOK, w.Code)
			suite.service.AssertExpectations(suite.T())
		})
	}
}

func (suite *OrderHandlerTestSuite) TestGetOrder_InvalidID() {
	w := suite.do(http.MethodGet, "/orders/invalid", nil, domain.RoleCustomer)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *OrderHandlerTestSuite) TestGetOrder_NotFound() {
	suite.service.On("GetOrder", suite.userID, false, suite.order.ID.String()).Return(nil, service.ErrOrderNotFound)

	w := suite.do(http.MethodGet, "/orders/"+suite.order.ID.String(), nil, domain.RoleCustomer)

	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *OrderHandlerTestSuite) TestCancelOrder_Success() {
	suite.service.On("CancelOrder", suite.userID, suite.order.ID.String()).Return(suite.order, nil)

	w := suite.do(http.MethodPost, "/orders/"+suite.order.ID.String()+"/cancel", nil, domain.RoleCustomer)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *OrderHandlerTestSuite) TestCancelOrder_InvalidTransition() {
	suite.service.On("CancelOrder", suite.userID, suite.order.ID.String()).Return(nil, service.ErrInvalidStatusTransition)

	w := suite.do(http.MethodPost, "/orders/"+suite.order.ID.String()+"/cancel", nil, domain.RoleCustomer)

	suite.Equal(http.StatusConflict, w.Code)
}

func (suite *OrderHandlerTestSuite) TestGetAllOrders_Success() {
	suite.service.On("GetAllOrders").Return([]domain.Order{*suite.order}, nil)

	w := suite.do(http.MethodGet, "/admin/orders", nil, domain.RoleAdmin)

	suite.Equal(http.StatusOK, w.Code)
	var resp []domain.Order
	suite.NoError(json.NewDecoder(w.Body).Decode(&resp))
	suite.Len(resp, 1)
}

func (suite *OrderHandlerTestSuite) TestGetAllOrders_InternalError() {
	suite.service.On("GetAllOrders").Return(nil, assert.AnError)

	w := suite.do(http.MethodGet, "/admin/orders", nil, domain.RoleAdmin)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *OrderHandlerTestSuite) TestUpdateOrderStatus_Success() {
	dto := request.UpdateOrderStatusRequestDTO{Status: "paid"}
	suite.service.On("UpdateOrderStatus", suite.order.ID.String(), dto).Return(suite.order, nil)

	w := suite.do(http.MethodPatch, "/admin/orders/"+suite.order.ID.String()+"/status", dto, domain.RoleAdmin)

	suite.Equal(http.StatusOK, w.Code)
}

func (suite *OrderHandlerTestSuite) TestUpdateOrderStatus_InvalidBody() {
	w := suite.do(http.MethodPatch, "/admin/orders/"+suite.order.ID.String()+"/status", "{invalid", domain.RoleAdmin)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *OrderHandlerTestSuite) TestUpdateOrderStatus_InvalidID() {
	w := suite.do(http.MethodPatch, "/admin/orders/invalid/status", request.UpdateOrderStatusRequestDTO{Status: "paid"}, domain.RoleAdmin)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *OrderHandlerTestSuite) TestUpdateOrderStatus_ErrorMapping() {
	cases := map[string]struct {
		err    error
		status int
	}{
		"invalid status":     {service.ErrInvalidOrderStatus, http.StatusBadRequest},
		"invalid transition": {service.ErrInvalidStatusTransition, http.StatusConflict},
		"not found":          {service.ErrOrderNotFound, http.StatusNotFound},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			suite.SetupTest()
			dto := request.UpdateOrderStatusRequestDTO{Status: "x"}
			suite.service.On("UpdateOrderStatus", suite.order.ID.String(), dto).Return(nil, tc.err)

			w := suite.do(http.MethodPatch, "/admin/orders/"+suite.order.ID.String()+"/status", dto, domain.RoleAdmin)

			suite.Equal(tc.status, w.Code)
		})
	}
}

func TestOrderHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(OrderHandlerTestSuite))
}
