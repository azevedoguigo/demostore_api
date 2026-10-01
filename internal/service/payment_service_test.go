package service_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) paymentResult(args mock.Arguments) (*domain.Payment, error) {
	p := args.Get(0)
	if p == nil {
		return nil, args.Error(1)
	}
	return p.(*domain.Payment), args.Error(1)
}

func (m *MockPaymentRepository) Create(payment *domain.Payment) error {
	return m.Called(payment).Error(0)
}

func (m *MockPaymentRepository) GetByOrderID(orderID uuid.UUID) (*domain.Payment, error) {
	return m.paymentResult(m.Called(orderID))
}

func (m *MockPaymentRepository) GetByProviderPaymentID(id string) (*domain.Payment, error) {
	return m.paymentResult(m.Called(id))
}

func (m *MockPaymentRepository) Update(payment *domain.Payment) error {
	return m.Called(payment).Error(0)
}

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) intentResult(args mock.Arguments) (*domain.PaymentIntent, error) {
	pi := args.Get(0)
	if pi == nil {
		return nil, args.Error(1)
	}
	return pi.(*domain.PaymentIntent), args.Error(1)
}

func (m *MockPaymentGateway) CreatePaymentIntent(orderID uuid.UUID, amount int64, currency string) (*domain.PaymentIntent, error) {
	return m.intentResult(m.Called(orderID, amount, currency))
}

func (m *MockPaymentGateway) GetPaymentIntent(id string) (*domain.PaymentIntent, error) {
	return m.intentResult(m.Called(id))
}

func (m *MockPaymentGateway) CancelPaymentIntent(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockPaymentGateway) RefundPaymentIntent(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockPaymentGateway) ParseWebhookEvent(payload []byte, signature string) (*domain.PaymentEvent, error) {
	args := m.Called(payload, signature)
	e := args.Get(0)
	if e == nil {
		return nil, args.Error(1)
	}
	return e.(*domain.PaymentEvent), args.Error(1)
}

type PaymentServiceTestSuite struct {
	suite.Suite
	repo      *MockPaymentRepository
	orderRepo *MockOrderRepository
	gateway   *MockPaymentGateway
	service   *service.PaymentServiceImpl
	userID    uuid.UUID
	order     *domain.Order
	payment   *domain.Payment
	intent    *domain.PaymentIntent
}

func (suite *PaymentServiceTestSuite) SetupTest() {
	suite.repo = new(MockPaymentRepository)
	suite.orderRepo = new(MockOrderRepository)
	suite.gateway = new(MockPaymentGateway)
	suite.service = service.NewPaymentService(suite.repo, suite.orderRepo, suite.gateway)
	suite.userID = uuid.New()

	suite.order = &domain.Order{
		ID:          uuid.New(),
		UserID:      suite.userID,
		Status:      domain.OrderStatusPending,
		TotalAmount: 4028,
		Currency:    domain.DefaultCurrency,
	}
	suite.payment = &domain.Payment{
		ID:                uuid.New(),
		OrderID:           suite.order.ID,
		ProviderPaymentID: "pi_123",
		Status:            domain.PaymentStatusPending,
		Amount:            suite.order.TotalAmount,
		Currency:          suite.order.Currency,
	}
	suite.intent = &domain.PaymentIntent{ID: "pi_123", ClientSecret: "pi_123_secret", Status: "requires_payment_method"}
}

func (suite *PaymentServiceTestSuite) webhook(eventType domain.PaymentEventType) {
	suite.gateway.On("ParseWebhookEvent", []byte("payload"), "sig").
		Return(&domain.PaymentEvent{Type: eventType, PaymentIntentID: "pi_123"}, nil)
	suite.repo.On("GetByProviderPaymentID", "pi_123").Return(suite.payment, nil)
}

// --- CreatePaymentIntent ---

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_New() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)
	suite.gateway.On("CreatePaymentIntent", suite.order.ID, int64(4028), "brl").Return(suite.intent, nil)
	suite.repo.On("Create", mock.MatchedBy(func(p *domain.Payment) bool {
		return p.OrderID == suite.order.ID && p.ProviderPaymentID == "pi_123" &&
			p.Status == domain.PaymentStatusPending && p.Amount == 4028 && p.Provider == "stripe" && p.ID != uuid.Nil
	})).Return(nil)

	resp, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal("pi_123_secret", resp.ClientSecret)
	suite.Equal(int64(4028), resp.Amount)
	suite.Equal(suite.order.ID, resp.OrderID)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_ReusesExisting() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.gateway.On("GetPaymentIntent", "pi_123").Return(suite.intent, nil)

	resp, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal(suite.payment.ID, resp.PaymentID)
	suite.Equal("pi_123_secret", resp.ClientSecret)
	suite.gateway.AssertNotCalled(suite.T(), "CreatePaymentIntent", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_ConcurrentCreateUsesStored() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound).Once()
	suite.gateway.On("CreatePaymentIntent", suite.order.ID, int64(4028), "brl").Return(suite.intent, nil)
	suite.repo.On("Create", mock.Anything).Return(assert.AnError)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil).Once()

	resp, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal(suite.payment.ID, resp.PaymentID)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_OtherUsersOrder() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	_, err := suite.service.CreatePaymentIntent(uuid.New(), suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderNotFound)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_OrderNotFound() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderNotFound)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_OrderNotPending() {
	for _, status := range []domain.OrderStatus{domain.OrderStatusPaid, domain.OrderStatusCancelled} {
		suite.SetupTest()
		suite.order.Status = status
		suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)

		_, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

		suite.ErrorIs(err, service.ErrOrderNotPayable, status)
	}
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_GatewayError() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)
	suite.gateway.On("CreatePaymentIntent", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrPaymentProvider)

	_, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrPaymentProvider)
	suite.repo.AssertNotCalled(suite.T(), "Create", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_InvalidID() {
	_, err := suite.service.CreatePaymentIntent(suite.userID, "invalid")

	suite.Error(err)
}

// --- CancelForOrder ---

func (suite *PaymentServiceTestSuite) TestCancelForOrder_NoPayment() {
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_CancelsOpenIntent() {
	for _, status := range []domain.PaymentStatus{domain.PaymentStatusPending, domain.PaymentStatusFailed} {
		suite.SetupTest()
		suite.payment.Status = status
		suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
		suite.gateway.On("CancelPaymentIntent", "pi_123").Return(nil)
		suite.repo.On("Update", suite.payment).Return(nil)

		suite.NoError(suite.service.CancelForOrder(suite.order.ID))
		suite.Equal(domain.PaymentStatusCancelled, suite.payment.Status)
	}
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_RefundsSucceededPayment() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.gateway.On("RefundPaymentIntent", "pi_123").Return(nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
	suite.Equal(domain.PaymentStatusRefunded, suite.payment.Status)
	suite.gateway.AssertNotCalled(suite.T(), "CancelPaymentIntent", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_AlreadyClosedIsNoop() {
	for _, status := range []domain.PaymentStatus{domain.PaymentStatusCancelled, domain.PaymentStatusRefunded} {
		suite.SetupTest()
		suite.payment.Status = status
		suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)

		suite.NoError(suite.service.CancelForOrder(suite.order.ID))
		suite.gateway.AssertNotCalled(suite.T(), "CancelPaymentIntent", mock.Anything)
		suite.gateway.AssertNotCalled(suite.T(), "RefundPaymentIntent", mock.Anything)
	}
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_IntentPaidMeanwhileBlocksCancel() {
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.gateway.On("CancelPaymentIntent", "pi_123").Return(domain.ErrPaymentIntentUnexpectedState)
	suite.gateway.On("GetPaymentIntent", "pi_123").Return(&domain.PaymentIntent{ID: "pi_123", Status: "succeeded"}, nil)

	err := suite.service.CancelForOrder(suite.order.ID)

	suite.ErrorIs(err, service.ErrPaymentAlreadyProcessed)
	suite.repo.AssertNotCalled(suite.T(), "Update", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_IntentAlreadyCanceledAtStripe() {
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.gateway.On("CancelPaymentIntent", "pi_123").Return(domain.ErrPaymentIntentUnexpectedState)
	suite.gateway.On("GetPaymentIntent", "pi_123").Return(&domain.PaymentIntent{ID: "pi_123", Status: "canceled"}, nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
	suite.Equal(domain.PaymentStatusCancelled, suite.payment.Status)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_GatewayError() {
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.gateway.On("CancelPaymentIntent", "pi_123").Return(domain.ErrPaymentProvider)

	suite.ErrorIs(suite.service.CancelForOrder(suite.order.ID), service.ErrPaymentProvider)
	suite.repo.AssertNotCalled(suite.T(), "Update", mock.Anything)
}

// --- HandleWebhook ---

func (suite *PaymentServiceTestSuite) TestWebhook_InvalidSignature() {
	suite.gateway.On("ParseWebhookEvent", []byte("payload"), "bad").Return(nil, domain.ErrInvalidWebhookSignature)

	err := suite.service.HandleWebhook([]byte("payload"), "bad")

	suite.ErrorIs(err, service.ErrInvalidWebhookSignature)
}

func (suite *PaymentServiceTestSuite) TestWebhook_IgnoresUnrelatedEvents() {
	suite.gateway.On("ParseWebhookEvent", []byte("payload"), "sig").Return(&domain.PaymentEvent{Type: "customer.created"}, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.repo.AssertNotCalled(suite.T(), "GetByProviderPaymentID", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_IgnoresUnknownIntent() {
	suite.gateway.On("ParseWebhookEvent", []byte("payload"), "sig").
		Return(&domain.PaymentEvent{Type: domain.PaymentEventSucceeded, PaymentIntentID: "pi_other"}, nil)
	suite.repo.On("GetByProviderPaymentID", "pi_other").Return(nil, gorm.ErrRecordNotFound)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededMarksOrderPaid() {
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.orderRepo.On("UpdateStatus", suite.order, domain.OrderStatusPending, false).Return(nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.OrderStatusPaid, suite.order.Status)
	suite.Equal(domain.PaymentStatusSucceeded, suite.payment.Status)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededIsIdempotent() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.webhook(domain.PaymentEventSucceeded)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.orderRepo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
	suite.repo.AssertNotCalled(suite.T(), "Update", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededRetryAfterOrderAlreadyPaid() {
	// Previous delivery updated the order but failed to save the payment.
	suite.order.Status = domain.OrderStatusPaid
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.PaymentStatusSucceeded, suite.payment.Status)
	suite.orderRepo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededForCancelledOrderRefunds() {
	suite.order.Status = domain.OrderStatusCancelled
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.gateway.On("RefundPaymentIntent", "pi_123").Return(nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.PaymentStatusRefunded, suite.payment.Status)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededRacingWithCancelRefunds() {
	suite.webhook(domain.PaymentEventSucceeded)
	cancelled := *suite.order
	cancelled.Status = domain.OrderStatusCancelled
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil).Once()
	suite.orderRepo.On("UpdateStatus", suite.order, domain.OrderStatusPending, false).Return(domain.ErrInvalidStatusTransition)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(&cancelled, nil).Once()
	suite.gateway.On("RefundPaymentIntent", "pi_123").Return(nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.PaymentStatusRefunded, suite.payment.Status)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededRefundFailureIsRetried() {
	suite.order.Status = domain.OrderStatusCancelled
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.gateway.On("RefundPaymentIntent", "pi_123").Return(domain.ErrPaymentProvider)

	err := suite.service.HandleWebhook([]byte("payload"), "sig")

	suite.ErrorIs(err, domain.ErrPaymentProvider)
	suite.Equal(domain.PaymentStatusPending, suite.payment.Status, "payment must stay pending so the retry refunds again")
}

func (suite *PaymentServiceTestSuite) TestWebhook_FailedKeepsOrderPending() {
	suite.webhook(domain.PaymentEventFailed)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.PaymentStatusFailed, suite.payment.Status)
	suite.orderRepo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_FailedAfterSuccessIsIgnored() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.webhook(domain.PaymentEventFailed)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.PaymentStatusSucceeded, suite.payment.Status)
}

func (suite *PaymentServiceTestSuite) TestWebhook_CanceledCancelsPendingOrder() {
	suite.webhook(domain.PaymentEventCanceled)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.orderRepo.On("UpdateStatus", suite.order, domain.OrderStatusPending, true).Return(nil)
	suite.repo.On("Update", suite.payment).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.OrderStatusCancelled, suite.order.Status)
	suite.Equal(domain.PaymentStatusCancelled, suite.payment.Status)
}

func (suite *PaymentServiceTestSuite) TestWebhook_CanceledAfterOurOwnCancelIsNoop() {
	suite.payment.Status = domain.PaymentStatusCancelled
	suite.order.Status = domain.OrderStatusCancelled
	suite.webhook(domain.PaymentEventCanceled)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.orderRepo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
	suite.repo.AssertNotCalled(suite.T(), "Update", mock.Anything)
}

func TestPaymentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentServiceTestSuite))
}
