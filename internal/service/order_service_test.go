package service_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateFromCart(order *domain.Order, cartID uuid.UUID) error {
	return m.Called(order, cartID).Error(0)
}

func (m *MockOrderRepository) GetByID(id uuid.UUID) (*domain.Order, error) {
	args := m.Called(id)
	o := args.Get(0)
	if o == nil {
		return nil, args.Error(1)
	}
	return o.(*domain.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByUserID(userID uuid.UUID) ([]domain.Order, error) {
	args := m.Called(userID)
	return args.Get(0).([]domain.Order), args.Error(1)
}

func (m *MockOrderRepository) GetAll() ([]domain.Order, error) {
	args := m.Called()
	return args.Get(0).([]domain.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateStatus(order *domain.Order, from domain.OrderStatus, restock bool) error {
	return m.Called(order, from, restock).Error(0)
}

type MockPaymentCanceler struct {
	mock.Mock
}

func (m *MockPaymentCanceler) CancelForOrder(orderID uuid.UUID) error {
	return m.Called(orderID).Error(0)
}

type OrderServiceTestSuite struct {
	suite.Suite
	repo     *MockOrderRepository
	cartRepo *MockCartRepository
	payments *MockPaymentCanceler
	service  *service.OrderServiceImpl
	userID   uuid.UUID
	cart     *domain.Cart
	order    *domain.Order
}

func (suite *OrderServiceTestSuite) SetupTest() {
	suite.repo = new(MockOrderRepository)
	suite.cartRepo = new(MockCartRepository)
	suite.payments = new(MockPaymentCanceler)
	suite.service = service.NewOrderService(suite.repo, suite.cartRepo, suite.payments)
	suite.userID = uuid.New()

	productA := &domain.Product{ID: uuid.New(), Name: "Product A", Price: 19.99}
	productB := &domain.Product{ID: uuid.New(), Name: "Product B", Price: 0.1}
	suite.cart = &domain.Cart{
		ID:     uuid.New(),
		UserID: suite.userID,
		Items: []domain.CartItem{
			{ProductID: productA.ID, Product: productA, Quantity: 2},
			{ProductID: productB.ID, Product: productB, Quantity: 3},
		},
	}

	suite.order = &domain.Order{
		ID:     uuid.New(),
		UserID: suite.userID,
		Status: domain.OrderStatusPending,
		Items:  []domain.OrderItem{{ProductID: productA.ID, Quantity: 2}},
	}
}

func (suite *OrderServiceTestSuite) TestCheckout_Success() {
	suite.cartRepo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("CreateFromCart", mock.AnythingOfType("*domain.Order"), suite.cart.ID).Return(nil)

	order, err := suite.service.Checkout(suite.userID)

	suite.NoError(err)
	suite.Equal(suite.userID, order.UserID)
	suite.Equal(domain.OrderStatusPending, order.Status)
	suite.Equal(domain.DefaultCurrency, order.Currency)
	suite.NotEqual(uuid.Nil, order.ID)
	suite.Require().Len(order.Items, 2)

	suite.Equal("Product A", order.Items[0].ProductName)
	suite.Equal(int64(1999), order.Items[0].UnitPrice)
	suite.Equal(int64(3998), order.Items[0].Subtotal)
	suite.Equal(order.ID, order.Items[0].OrderID)
	suite.NotEqual(uuid.Nil, order.Items[0].ID)
	suite.Equal(int64(30), order.Items[1].Subtotal)
	suite.Equal(int64(4028), order.TotalAmount)
}

func (suite *OrderServiceTestSuite) TestCheckout_NoCart() {
	suite.cartRepo.On("GetByUserID", suite.userID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.Checkout(suite.userID)

	suite.ErrorIs(err, service.ErrEmptyCart)
}

func (suite *OrderServiceTestSuite) TestCheckout_EmptyCart() {
	suite.cartRepo.On("GetByUserID", suite.userID).Return(&domain.Cart{ID: uuid.New()}, nil)

	_, err := suite.service.Checkout(suite.userID)

	suite.ErrorIs(err, service.ErrEmptyCart)
	suite.repo.AssertNotCalled(suite.T(), "CreateFromCart", mock.Anything, mock.Anything)
}

func (suite *OrderServiceTestSuite) TestCheckout_DeletedProduct() {
	suite.cart.Items[1].Product = nil
	suite.cartRepo.On("GetByUserID", suite.userID).Return(suite.cart, nil)

	_, err := suite.service.Checkout(suite.userID)

	suite.ErrorIs(err, service.ErrProductNotFound)
}

func (suite *OrderServiceTestSuite) TestCheckout_InsufficientStock() {
	suite.cartRepo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("CreateFromCart", mock.Anything, suite.cart.ID).Return(domain.ErrInsufficientStock)

	_, err := suite.service.Checkout(suite.userID)

	suite.ErrorIs(err, service.ErrInsufficientStock)
}

func (suite *OrderServiceTestSuite) TestCheckout_CartRepositoryError() {
	suite.cartRepo.On("GetByUserID", suite.userID).Return(nil, assert.AnError)

	_, err := suite.service.Checkout(suite.userID)

	suite.ErrorIs(err, assert.AnError)
}

func (suite *OrderServiceTestSuite) TestGetUserOrders() {
	suite.repo.On("GetByUserID", suite.userID).Return([]domain.Order{*suite.order}, nil)

	orders, err := suite.service.GetUserOrders(suite.userID)

	suite.NoError(err)
	suite.Len(orders, 1)
}

func (suite *OrderServiceTestSuite) TestGetAllOrders() {
	suite.repo.On("GetAll").Return([]domain.Order{*suite.order}, nil)

	orders, err := suite.service.GetAllOrders()

	suite.NoError(err)
	suite.Len(orders, 1)
}

func (suite *OrderServiceTestSuite) TestGetOrder_Owner() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	order, err := suite.service.GetOrder(suite.userID, false, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal(suite.order, order)
}

func (suite *OrderServiceTestSuite) TestGetOrder_OtherUserIsNotFound() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	_, err := suite.service.GetOrder(uuid.New(), false, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderNotFound)
}

func (suite *OrderServiceTestSuite) TestGetOrder_AdminSeesAnyOrder() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	order, err := suite.service.GetOrder(uuid.New(), true, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal(suite.order, order)
}

func (suite *OrderServiceTestSuite) TestGetOrder_NotFound() {
	suite.repo.On("GetByID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.GetOrder(suite.userID, false, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderNotFound)
}

func (suite *OrderServiceTestSuite) TestGetOrder_InvalidID() {
	_, err := suite.service.GetOrder(suite.userID, false, "invalid")

	suite.Error(err)
}

func (suite *OrderServiceTestSuite) TestCancelOrder_PendingRestoresStock() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.payments.On("CancelForOrder", suite.order.ID).Return(nil)
	suite.repo.On("UpdateStatus", suite.order, domain.OrderStatusPending, true).Return(nil)

	order, err := suite.service.CancelOrder(suite.userID, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal(domain.OrderStatusCancelled, order.Status)
	suite.repo.AssertExpectations(suite.T())
	suite.payments.AssertExpectations(suite.T())
}

func (suite *OrderServiceTestSuite) TestCancelOrder_PaymentCancelFailureKeepsOrder() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.payments.On("CancelForOrder", suite.order.ID).Return(service.ErrPaymentAlreadyProcessed)

	_, err := suite.service.CancelOrder(suite.userID, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrPaymentAlreadyProcessed)
	suite.Equal(domain.OrderStatusPending, suite.order.Status)
	suite.repo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *OrderServiceTestSuite) TestCancelOrder_PaidCannotBeCancelledByCustomer() {
	suite.order.Status = domain.OrderStatusPaid
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	_, err := suite.service.CancelOrder(suite.userID, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrInvalidStatusTransition)
	suite.repo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *OrderServiceTestSuite) TestCancelOrder_OtherUser() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	_, err := suite.service.CancelOrder(uuid.New(), suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderNotFound)
}

func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_Success() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("UpdateStatus", suite.order, domain.OrderStatusPending, false).Return(nil)

	order, err := suite.service.UpdateOrderStatus(suite.order.ID.String(), request.UpdateOrderStatusRequestDTO{Status: "paid"})

	suite.NoError(err)
	suite.Equal(domain.OrderStatusPaid, order.Status)
	suite.payments.AssertNotCalled(suite.T(), "CancelForOrder", mock.Anything)
}

func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_AdminCancelPaidRestoresStock() {
	suite.order.Status = domain.OrderStatusPaid
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.payments.On("CancelForOrder", suite.order.ID).Return(nil)
	suite.repo.On("UpdateStatus", suite.order, domain.OrderStatusPaid, true).Return(nil)

	_, err := suite.service.UpdateOrderStatus(suite.order.ID.String(), request.UpdateOrderStatusRequestDTO{Status: "cancelled"})

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_InvalidStatus() {
	_, err := suite.service.UpdateOrderStatus(suite.order.ID.String(), request.UpdateOrderStatusRequestDTO{Status: "refunded"})

	suite.ErrorIs(err, service.ErrInvalidOrderStatus)
}

func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_InvalidTransition() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	_, err := suite.service.UpdateOrderStatus(suite.order.ID.String(), request.UpdateOrderStatusRequestDTO{Status: "delivered"})

	suite.ErrorIs(err, service.ErrInvalidStatusTransition)
	suite.Equal(domain.OrderStatusPending, suite.order.Status, "status must not change on rejected transition")
}

func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_ConcurrentChange() {
	suite.repo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("UpdateStatus", suite.order, domain.OrderStatusPending, false).Return(domain.ErrInvalidStatusTransition)

	_, err := suite.service.UpdateOrderStatus(suite.order.ID.String(), request.UpdateOrderStatusRequestDTO{Status: "paid"})

	suite.ErrorIs(err, service.ErrInvalidStatusTransition)
}

func TestOrderServiceTestSuite(t *testing.T) {
	suite.Run(t, new(OrderServiceTestSuite))
}
