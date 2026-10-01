package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
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

type MockRefundRepository struct {
	mock.Mock
}

func (m *MockRefundRepository) refundResult(args mock.Arguments) (*domain.Refund, error) {
	r := args.Get(0)
	if r == nil {
		return nil, args.Error(1)
	}
	return r.(*domain.Refund), args.Error(1)
}

func (m *MockRefundRepository) Reserve(refund *domain.Refund) error {
	return m.Called(refund).Error(0)
}

func (m *MockRefundRepository) SetProviderRefundID(id uuid.UUID, providerRefundID string) error {
	return m.Called(id, providerRefundID).Error(0)
}

func (m *MockRefundRepository) MarkSucceeded(refund *domain.Refund) (bool, error) {
	args := m.Called(refund)
	return args.Bool(0), args.Error(1)
}

func (m *MockRefundRepository) MarkFailed(refund *domain.Refund) (bool, error) {
	args := m.Called(refund)
	return args.Bool(0), args.Error(1)
}

func (m *MockRefundRepository) GetByID(id uuid.UUID) (*domain.Refund, error) {
	return m.refundResult(m.Called(id))
}

func (m *MockRefundRepository) GetByProviderRefundID(providerRefundID string) (*domain.Refund, error) {
	return m.refundResult(m.Called(providerRefundID))
}

func (m *MockRefundRepository) GetByOrderID(orderID uuid.UUID) ([]domain.Refund, error) {
	args := m.Called(orderID)
	return args.Get(0).([]domain.Refund), args.Error(1)
}

func (m *MockRefundRepository) GetPending(paymentID uuid.UUID) ([]domain.Refund, error) {
	args := m.Called(paymentID)
	return args.Get(0).([]domain.Refund), args.Error(1)
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

func (m *MockPaymentGateway) CreatePaymentIntent(req domain.PaymentIntentRequest) (*domain.PaymentIntent, error) {
	return m.intentResult(m.Called(req))
}

func (m *MockPaymentGateway) GetPaymentIntent(id string) (*domain.PaymentIntent, error) {
	return m.intentResult(m.Called(id))
}

func (m *MockPaymentGateway) CancelPaymentIntent(id string) error {
	return m.Called(id).Error(0)
}

func (m *MockPaymentGateway) GetPaymentMethodType(paymentIntentID string) (string, error) {
	args := m.Called(paymentIntentID)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) RefundPaymentIntent(paymentIntentID string, amount int64, refundID uuid.UUID) (*domain.ProviderRefund, error) {
	args := m.Called(paymentIntentID, amount, refundID)
	r := args.Get(0)
	if r == nil {
		return nil, args.Error(1)
	}
	return r.(*domain.ProviderRefund), args.Error(1)
}

func (m *MockPaymentGateway) ParseWebhookEvent(payload []byte, signature string) (*domain.PaymentEvent, error) {
	args := m.Called(payload, signature)
	e := args.Get(0)
	if e == nil {
		return nil, args.Error(1)
	}
	return e.(*domain.PaymentEvent), args.Error(1)
}

// markRefund mimics the repository settling a pending refund.
func markRefund(status domain.RefundStatus) func(mock.Arguments) {
	return func(args mock.Arguments) {
		args.Get(0).(*domain.Refund).Status = status
	}
}

type PaymentServiceTestSuite struct {
	suite.Suite
	repo       *MockPaymentRepository
	orderRepo  *MockOrderRepository
	refundRepo *MockRefundRepository
	gateway    *MockPaymentGateway
	service    *service.PaymentServiceImpl
	userID     uuid.UUID
	order      *domain.Order
	payment    *domain.Payment
	intent     *domain.PaymentIntent
	expiresAt  time.Time
}

func (suite *PaymentServiceTestSuite) SetupTest() {
	suite.repo = new(MockPaymentRepository)
	suite.orderRepo = new(MockOrderRepository)
	suite.refundRepo = new(MockRefundRepository)
	suite.gateway = new(MockPaymentGateway)
	suite.service = service.NewPaymentService(suite.repo, suite.orderRepo, suite.refundRepo, suite.gateway)
	suite.userID = uuid.New()
	suite.expiresAt = time.Now().Add(20 * time.Minute)

	suite.order = &domain.Order{
		ID:          uuid.New(),
		UserID:      suite.userID,
		Status:      domain.OrderStatusPending,
		TotalAmount: 4028,
		Currency:    domain.DefaultCurrency,
		ExpiresAt:   &suite.expiresAt,
		Items: []domain.OrderItem{
			{ID: uuid.New(), ProductID: uuid.New(), UnitPrice: 1999, Quantity: 2, Subtotal: 3998},
			{ID: uuid.New(), ProductID: uuid.New(), UnitPrice: 10, Quantity: 3, Subtotal: 30},
		},
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
	suite.webhookEvent(&domain.PaymentEvent{Type: eventType, PaymentIntentID: "pi_123"})
}

func (suite *PaymentServiceTestSuite) webhookEvent(event *domain.PaymentEvent) {
	suite.gateway.On("ParseWebhookEvent", []byte("payload"), "sig").Return(event, nil)
	if event.PaymentIntentID != "" {
		suite.repo.On("GetByProviderPaymentID", event.PaymentIntentID).Return(suite.payment, nil)
	}
}

// expectRefund sets up a refund of amount for a card payment that Stripe settles with providerStatus.
func (suite *PaymentServiceTestSuite) expectRefund(amount int64, providerStatus string) {
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodCard, nil)
	suite.refundRepo.On("Reserve", mock.MatchedBy(func(r *domain.Refund) bool { return r.Amount == amount })).Return(nil)
	suite.gateway.On("RefundPaymentIntent", "pi_123", amount, mock.Anything).
		Return(&domain.ProviderRefund{ID: "re_1", Status: providerStatus}, nil)
	suite.refundRepo.On("SetProviderRefundID", mock.Anything, "re_1").Return(nil)
}

// --- CreatePaymentIntent ---

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_New() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)
	suite.gateway.On("CreatePaymentIntent", domain.PaymentIntentRequest{
		OrderID:            suite.order.ID,
		Amount:             4028,
		Currency:           "brl",
		PaymentMethodTypes: []string{"card", "pix", "boleto"},
		ExpiresAt:          suite.expiresAt,
	}).Return(suite.intent, nil)
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
	suite.gateway.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_LegacyOrderWithoutExpiration() {
	suite.order.ExpiresAt = nil
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)
	suite.gateway.On("CreatePaymentIntent", mock.MatchedBy(func(req domain.PaymentIntentRequest) bool {
		return req.ExpiresAt.IsZero()
	})).Return(suite.intent, nil)
	suite.repo.On("Create", mock.Anything).Return(nil)

	_, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.NoError(err)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_ReusesExisting() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.gateway.On("GetPaymentIntent", "pi_123").Return(suite.intent, nil)

	resp, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.NoError(err)
	suite.Equal(suite.payment.ID, resp.PaymentID)
	suite.Equal("pi_123_secret", resp.ClientSecret)
	suite.gateway.AssertNotCalled(suite.T(), "CreatePaymentIntent", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_ConcurrentCreateUsesStored() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound).Once()
	suite.gateway.On("CreatePaymentIntent", mock.Anything).Return(suite.intent, nil)
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

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_OrderExpired() {
	past := time.Now().Add(-time.Second)
	suite.order.ExpiresAt = &past
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	_, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderExpired)
	suite.repo.AssertNotCalled(suite.T(), "GetByOrderID", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_GatewayError() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)
	suite.gateway.On("CreatePaymentIntent", mock.Anything).Return(nil, domain.ErrPaymentProvider)

	_, err := suite.service.CreatePaymentIntent(suite.userID, suite.order.ID.String())

	suite.ErrorIs(err, service.ErrPaymentProvider)
	suite.repo.AssertNotCalled(suite.T(), "Create", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCreatePaymentIntent_InvalidID() {
	_, err := suite.service.CreatePaymentIntent(suite.userID, "invalid")

	suite.Error(err)
}

// --- CancelForOrder: open intents ---

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

func (suite *PaymentServiceTestSuite) TestCancelForOrder_AlreadyClosedIsNoop() {
	for _, status := range []domain.PaymentStatus{domain.PaymentStatusCancelled, domain.PaymentStatusRefunded} {
		suite.SetupTest()
		suite.payment.Status = status
		suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)

		suite.NoError(suite.service.CancelForOrder(suite.order.ID))
		suite.gateway.AssertNotCalled(suite.T(), "CancelPaymentIntent", mock.Anything)
		suite.gateway.AssertNotCalled(suite.T(), "RefundPaymentIntent", mock.Anything, mock.Anything, mock.Anything)
	}
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_PaymentUnderWayBlocksCancel() {
	// "succeeded": paid, webhook not arrived yet. "requires_action": boleto issued, can't be cancelled before it expires.
	for _, status := range []string{"succeeded", "processing", "requires_action"} {
		suite.SetupTest()
		suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
		suite.gateway.On("CancelPaymentIntent", "pi_123").Return(domain.ErrPaymentIntentUnexpectedState)
		suite.gateway.On("GetPaymentIntent", "pi_123").Return(&domain.PaymentIntent{ID: "pi_123", Status: status}, nil)

		err := suite.service.CancelForOrder(suite.order.ID)

		suite.ErrorIs(err, service.ErrPaymentInProgress, status)
		suite.repo.AssertNotCalled(suite.T(), "Update", mock.Anything)
	}
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

// --- CancelForOrder: captured payments (refund of the remaining amount) ---

func (suite *PaymentServiceTestSuite) TestCancelForOrder_RefundsSucceededPayment() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.expectRefund(4028, domain.ProviderRefundSucceeded)
	suite.refundRepo.On("MarkSucceeded", mock.MatchedBy(func(r *domain.Refund) bool {
		return !r.Restock && len(r.Items) == 0 && r.PaymentID == suite.payment.ID && *r.ProviderRefundID == "re_1"
	})).Return(true, nil)

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
	suite.refundRepo.AssertExpectations(suite.T())
	suite.gateway.AssertNotCalled(suite.T(), "CancelPaymentIntent", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_RefundsOnlyTheRemainingAmount() {
	suite.payment.Status = domain.PaymentStatusPartiallyRefunded
	suite.payment.RefundedAmount = 1999
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.expectRefund(2029, domain.ProviderRefundSucceeded)
	suite.refundRepo.On("MarkSucceeded", mock.Anything).Return(true, nil)

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
	suite.gateway.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_AsyncRefundStaysPending() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.expectRefund(4028, "pending")

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
	suite.refundRepo.AssertNotCalled(suite.T(), "MarkSucceeded", mock.Anything)
	suite.refundRepo.AssertNotCalled(suite.T(), "MarkFailed", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_BoletoCantBeRefunded() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodBoleto, nil)

	err := suite.service.CancelForOrder(suite.order.ID)

	suite.ErrorIs(err, service.ErrRefundNotSupported)
	suite.refundRepo.AssertNotCalled(suite.T(), "Reserve", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_PendingRefundBlocksCancel() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	sent := "re_0"
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).
		Return([]domain.Refund{{ID: uuid.New(), Status: domain.RefundStatusPending, ProviderRefundID: &sent}}, nil)

	err := suite.service.CancelForOrder(suite.order.ID)

	suite.ErrorIs(err, service.ErrRefundInProgress)
	suite.gateway.AssertNotCalled(suite.T(), "RefundPaymentIntent", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_ResendsUnsentRefundThenRefundsRest() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	unsent := domain.Refund{ID: uuid.New(), PaymentID: suite.payment.ID, Amount: 1999, Status: domain.RefundStatusPending}
	settled := *suite.payment
	settled.Status = domain.PaymentStatusPartiallyRefunded
	settled.RefundedAmount = 1999

	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil).Once()
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{unsent}, nil)
	// The unsent refund is resent with its own ID, so Stripe's idempotency avoids a duplicate.
	suite.gateway.On("RefundPaymentIntent", "pi_123", int64(1999), unsent.ID).
		Return(&domain.ProviderRefund{ID: "re_0", Status: domain.ProviderRefundSucceeded}, nil)
	suite.refundRepo.On("SetProviderRefundID", unsent.ID, "re_0").Return(nil)
	suite.refundRepo.On("MarkSucceeded", mock.MatchedBy(func(r *domain.Refund) bool { return r.ID == unsent.ID })).
		Run(markRefund(domain.RefundStatusSucceeded)).Return(true, nil).Once()
	suite.repo.On("GetByOrderID", suite.order.ID).Return(&settled, nil).Once()
	suite.expectRefund(2029, domain.ProviderRefundSucceeded)
	suite.refundRepo.On("MarkSucceeded", mock.Anything).Return(true, nil).Once()

	suite.NoError(suite.service.CancelForOrder(suite.order.ID))
	suite.gateway.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_RejectedRefundIsReleased() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodCard, nil)
	suite.refundRepo.On("Reserve", mock.Anything).Return(nil)
	suite.gateway.On("RefundPaymentIntent", "pi_123", int64(4028), mock.Anything).
		Return(nil, errors.Join(domain.ErrPaymentProvider, domain.ErrPaymentProviderRejected))
	suite.refundRepo.On("MarkFailed", mock.Anything).Return(true, nil)

	err := suite.service.CancelForOrder(suite.order.ID)

	suite.ErrorIs(err, service.ErrPaymentProvider)
	suite.refundRepo.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestCancelForOrder_UnknownRefundOutcomeKeepsReservation() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodCard, nil)
	suite.refundRepo.On("Reserve", mock.Anything).Return(nil)
	suite.gateway.On("RefundPaymentIntent", "pi_123", int64(4028), mock.Anything).Return(nil, domain.ErrPaymentProvider)

	err := suite.service.CancelForOrder(suite.order.ID)

	suite.ErrorIs(err, service.ErrPaymentProvider)
	suite.refundRepo.AssertNotCalled(suite.T(), "MarkFailed", mock.Anything)
}

// --- HandleWebhook: payment intents ---

func (suite *PaymentServiceTestSuite) TestWebhook_InvalidSignature() {
	suite.gateway.On("ParseWebhookEvent", []byte("payload"), "bad").Return(nil, domain.ErrInvalidWebhookSignature)

	err := suite.service.HandleWebhook([]byte("payload"), "bad")

	suite.ErrorIs(err, service.ErrInvalidWebhookSignature)
}

func (suite *PaymentServiceTestSuite) TestWebhook_IgnoresUnrelatedEvents() {
	suite.webhookEvent(&domain.PaymentEvent{Type: "customer.created"})

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
	suite.order.Status = domain.OrderStatusPaid
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.orderRepo.AssertNotCalled(suite.T(), "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
	suite.repo.AssertNotCalled(suite.T(), "Update", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededIgnoredOnceRefunded() {
	for _, status := range []domain.PaymentStatus{domain.PaymentStatusRefunded, domain.PaymentStatusPartiallyRefunded} {
		suite.SetupTest()
		suite.payment.Status = status
		suite.webhook(domain.PaymentEventSucceeded)

		suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
		suite.orderRepo.AssertNotCalled(suite.T(), "GetByID", mock.Anything)
	}
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
	suite.repo.On("Update", suite.payment).Return(nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.expectRefund(4028, domain.ProviderRefundSucceeded)
	suite.refundRepo.On("MarkSucceeded", mock.Anything).Return(true, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.Equal(domain.PaymentStatusSucceeded, suite.payment.Status)
	suite.refundRepo.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededRacingWithCancelRefunds() {
	suite.webhook(domain.PaymentEventSucceeded)
	cancelled := *suite.order
	cancelled.Status = domain.OrderStatusCancelled
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil).Once()
	suite.orderRepo.On("UpdateStatus", suite.order, domain.OrderStatusPending, false).Return(domain.ErrInvalidStatusTransition)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(&cancelled, nil).Once()
	suite.repo.On("Update", suite.payment).Return(nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.expectRefund(4028, domain.ProviderRefundSucceeded)
	suite.refundRepo.On("MarkSucceeded", mock.Anything).Return(true, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.gateway.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededForCancelledOrderWithRefundInFlight() {
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.order.Status = domain.OrderStatusCancelled
	sent := "re_1"
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).
		Return([]domain.Refund{{ID: uuid.New(), Status: domain.RefundStatusPending, ProviderRefundID: &sent}}, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.gateway.AssertNotCalled(suite.T(), "RefundPaymentIntent", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededBoletoForCancelledOrderNeedsManualRefund() {
	suite.order.Status = domain.OrderStatusCancelled
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("Update", suite.payment).Return(nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodBoleto, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"), "must not make Stripe retry forever")
	suite.Equal(domain.PaymentStatusSucceeded, suite.payment.Status)
	suite.refundRepo.AssertNotCalled(suite.T(), "Reserve", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_SucceededRefundFailureIsRetried() {
	suite.order.Status = domain.OrderStatusCancelled
	suite.webhook(domain.PaymentEventSucceeded)
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("Update", suite.payment).Return(nil)
	suite.refundRepo.On("GetPending", suite.payment.ID).Return([]domain.Refund{}, nil)
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return("", domain.ErrPaymentProvider)

	err := suite.service.HandleWebhook([]byte("payload"), "sig")

	suite.ErrorIs(err, domain.ErrPaymentProvider)
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

func (suite *PaymentServiceTestSuite) TestWebhook_BoletoIssuedExtendsOrderExpiration() {
	boletoExpiresAt := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	suite.webhookEvent(&domain.PaymentEvent{
		Type:                domain.PaymentEventRequiresAction,
		PaymentIntentID:     "pi_123",
		NextActionType:      domain.NextActionBoletoDisplayDetails,
		NextActionExpiresAt: boletoExpiresAt,
	})
	suite.orderRepo.On("ExtendExpiration", suite.order.ID, boletoExpiresAt.Add(domain.BoletoConfirmationGrace)).Return(nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.orderRepo.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestWebhook_OtherRequiredActionsDontExtendExpiration() {
	suite.webhookEvent(&domain.PaymentEvent{
		Type:            domain.PaymentEventRequiresAction,
		PaymentIntentID: "pi_123",
		NextActionType:  "use_stripe_sdk",
	})

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.orderRepo.AssertNotCalled(suite.T(), "ExtendExpiration", mock.Anything, mock.Anything)
}

// --- HandleWebhook: refunds ---

func (suite *PaymentServiceTestSuite) TestWebhook_RefundSucceededMatchedByLocalID() {
	refund := &domain.Refund{ID: uuid.New(), Status: domain.RefundStatusPending}
	suite.webhookEvent(&domain.PaymentEvent{
		Type:   domain.PaymentEventRefundUpdated,
		Refund: &domain.ProviderRefund{ID: "re_1", Status: "succeeded", LocalRefundID: refund.ID.String()},
	})
	suite.refundRepo.On("GetByID", refund.ID).Return(refund, nil)
	suite.refundRepo.On("SetProviderRefundID", refund.ID, "re_1").Return(nil)
	suite.refundRepo.On("MarkSucceeded", refund).Return(true, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.refundRepo.AssertExpectations(suite.T())
}

func (suite *PaymentServiceTestSuite) TestWebhook_RefundMatchedByProviderID() {
	sent := "re_1"
	refund := &domain.Refund{ID: uuid.New(), Status: domain.RefundStatusPending, ProviderRefundID: &sent}
	suite.webhookEvent(&domain.PaymentEvent{
		Type:   domain.PaymentEventRefundFailed,
		Refund: &domain.ProviderRefund{ID: "re_1", Status: "failed"},
	})
	suite.refundRepo.On("GetByProviderRefundID", "re_1").Return(refund, nil)
	suite.refundRepo.On("MarkFailed", refund).Return(true, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.refundRepo.AssertNotCalled(suite.T(), "SetProviderRefundID", mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_RefundStillPendingChangesNothing() {
	sent := "re_1"
	refund := &domain.Refund{ID: uuid.New(), Status: domain.RefundStatusPending, ProviderRefundID: &sent}
	suite.webhookEvent(&domain.PaymentEvent{
		Type:   domain.PaymentEventRefundCreated,
		Refund: &domain.ProviderRefund{ID: "re_1", Status: "pending", LocalRefundID: refund.ID.String()},
	})
	suite.refundRepo.On("GetByID", refund.ID).Return(refund, nil)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
	suite.refundRepo.AssertNotCalled(suite.T(), "MarkSucceeded", mock.Anything)
	suite.refundRepo.AssertNotCalled(suite.T(), "MarkFailed", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestWebhook_UnknownRefundIsIgnored() {
	suite.webhookEvent(&domain.PaymentEvent{
		Type:   domain.PaymentEventRefundUpdated,
		Refund: &domain.ProviderRefund{ID: "re_dashboard", Status: "succeeded"},
	})
	suite.refundRepo.On("GetByProviderRefundID", "re_dashboard").Return(nil, gorm.ErrRecordNotFound)

	suite.NoError(suite.service.HandleWebhook([]byte("payload"), "sig"))
}

// --- RefundItems ---

func (suite *PaymentServiceTestSuite) paidOrder() {
	suite.order.Status = domain.OrderStatusDelivered
	suite.payment.Status = domain.PaymentStatusSucceeded
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(suite.payment, nil)
}

func (suite *PaymentServiceTestSuite) refundDTO(items ...request.RefundItemRequestDTO) request.CreateRefundRequestDTO {
	return request.CreateRefundRequestDTO{Items: items, Reason: "damaged"}
}

func (suite *PaymentServiceTestSuite) TestRefundItems_Success() {
	suite.paidOrder()
	productA := suite.order.Items[0]
	suite.expectRefund(1999, domain.ProviderRefundSucceeded)
	suite.refundRepo.On("MarkSucceeded", mock.Anything).Run(markRefund(domain.RefundStatusSucceeded)).Return(true, nil)

	refund, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(
		request.RefundItemRequestDTO{ProductID: productA.ProductID.String(), Quantity: 1},
	))

	suite.NoError(err)
	suite.Equal(int64(1999), refund.Amount)
	suite.Equal("damaged", refund.Reason)
	suite.True(refund.Restock)
	suite.Equal(domain.RefundStatusSucceeded, refund.Status)
	suite.Require().Len(refund.Items, 1)
	suite.Equal(productA.ID, refund.Items[0].OrderItemID)
	suite.Equal(1, refund.Items[0].Quantity)
	suite.Equal(refund.ID, refund.Items[0].RefundID)
}

func (suite *PaymentServiceTestSuite) TestRefundItems_MergesRepeatedProducts() {
	suite.paidOrder()
	productB := suite.order.Items[1]
	suite.expectRefund(30, "pending")

	refund, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(
		request.RefundItemRequestDTO{ProductID: productB.ProductID.String(), Quantity: 1},
		request.RefundItemRequestDTO{ProductID: productB.ProductID.String(), Quantity: 2},
	))

	suite.NoError(err)
	suite.Require().Len(refund.Items, 1)
	suite.Equal(3, refund.Items[0].Quantity)
	suite.Equal(domain.RefundStatusPending, refund.Status)
	suite.refundRepo.AssertNotCalled(suite.T(), "MarkSucceeded", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestRefundItems_InvalidItems() {
	productA := suite.order.Items[0].ProductID.String()
	cases := map[string][]request.RefundItemRequestDTO{
		"no items":            nil,
		"invalid product id":  {{ProductID: "x", Quantity: 1}},
		"product not ordered": {{ProductID: uuid.New().String(), Quantity: 1}},
		"zero quantity":       {{ProductID: productA, Quantity: 0}},
	}

	for name, items := range cases {
		suite.Run(name, func() {
			suite.SetupTest()
			suite.paidOrder()

			_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(items...))

			suite.ErrorIs(err, service.ErrInvalidRefundItems)
			suite.refundRepo.AssertNotCalled(suite.T(), "Reserve", mock.Anything)
		})
	}
}

func (suite *PaymentServiceTestSuite) TestRefundItems_QuantityExceeded() {
	suite.order.Items[0].RefundedQuantity = 1
	suite.paidOrder()

	_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(
		request.RefundItemRequestDTO{ProductID: suite.order.Items[0].ProductID.String(), Quantity: 2},
	))

	suite.ErrorIs(err, service.ErrRefundQuantityExceeded)
	suite.refundRepo.AssertNotCalled(suite.T(), "Reserve", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestRefundItems_ConcurrentReservationFails() {
	suite.paidOrder()
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodCard, nil)
	suite.refundRepo.On("Reserve", mock.Anything).Return(domain.ErrRefundQuantityExceeded)

	_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(
		request.RefundItemRequestDTO{ProductID: suite.order.Items[0].ProductID.String(), Quantity: 2},
	))

	suite.ErrorIs(err, service.ErrRefundQuantityExceeded)
	suite.gateway.AssertNotCalled(suite.T(), "RefundPaymentIntent", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestRefundItems_Boleto() {
	suite.paidOrder()
	suite.gateway.On("GetPaymentMethodType", "pi_123").Return(domain.PaymentMethodBoleto, nil)

	_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(
		request.RefundItemRequestDTO{ProductID: suite.order.Items[0].ProductID.String(), Quantity: 1},
	))

	suite.ErrorIs(err, service.ErrRefundNotSupported)
	suite.refundRepo.AssertNotCalled(suite.T(), "Reserve", mock.Anything)
}

func (suite *PaymentServiceTestSuite) TestRefundItems_OrderNotRefundable() {
	item := request.RefundItemRequestDTO{ProductID: suite.order.Items[0].ProductID.String(), Quantity: 1}

	for _, status := range []domain.OrderStatus{domain.OrderStatusPending, domain.OrderStatusCancelled} {
		suite.SetupTest()
		suite.order.Status = status
		suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)

		_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(item))

		suite.ErrorIs(err, service.ErrOrderNotRefundable, status)
	}

	suite.SetupTest()
	suite.order.Status = domain.OrderStatusPaid
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.repo.On("GetByOrderID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)
	_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(item))
	suite.ErrorIs(err, service.ErrOrderNotRefundable, "paid manually, no payment")

	suite.SetupTest()
	suite.paidOrder()
	suite.payment.Status = domain.PaymentStatusRefunded
	_, err = suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO(item))
	suite.ErrorIs(err, service.ErrOrderNotRefundable, "fully refunded")
}

func (suite *PaymentServiceTestSuite) TestRefundItems_OrderNotFound() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.RefundItems(suite.order.ID.String(), suite.refundDTO())

	suite.ErrorIs(err, service.ErrOrderNotFound)

	_, err = suite.service.RefundItems("invalid", suite.refundDTO())
	suite.Error(err)
}

func (suite *PaymentServiceTestSuite) TestGetOrderRefunds() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(suite.order, nil)
	suite.refundRepo.On("GetByOrderID", suite.order.ID).Return([]domain.Refund{{ID: uuid.New()}}, nil)

	refunds, err := suite.service.GetOrderRefunds(suite.order.ID.String())

	suite.NoError(err)
	suite.Len(refunds, 1)
}

func (suite *PaymentServiceTestSuite) TestGetOrderRefunds_OrderNotFound() {
	suite.orderRepo.On("GetByID", suite.order.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.GetOrderRefunds(suite.order.ID.String())

	suite.ErrorIs(err, service.ErrOrderNotFound)
}

func TestPaymentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentServiceTestSuite))
}
