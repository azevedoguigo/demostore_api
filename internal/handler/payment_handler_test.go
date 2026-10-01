package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) CreatePaymentIntent(userID uuid.UUID, orderID string) (*response.PaymentIntentResponse, error) {
	args := m.Called(userID, orderID)
	r := args.Get(0)
	if r == nil {
		return nil, args.Error(1)
	}
	return r.(*response.PaymentIntentResponse), args.Error(1)
}

func (m *MockPaymentService) HandleWebhook(payload []byte, signature string) error {
	return m.Called(payload, signature).Error(0)
}

func (m *MockPaymentService) CancelForOrder(orderID uuid.UUID) error {
	return m.Called(orderID).Error(0)
}

type PaymentHandlerTestSuite struct {
	suite.Suite
	service *MockPaymentService
	router  *chi.Mux
	userID  uuid.UUID
	orderID string
}

func (suite *PaymentHandlerTestSuite) SetupTest() {
	suite.service = new(MockPaymentService)
	h := handler.NewPaymentHandler(suite.service)
	suite.userID = uuid.New()
	suite.orderID = uuid.New().String()

	suite.router = chi.NewRouter()
	suite.router.Post("/orders/{id}/payment", h.CreatePaymentIntent)
	suite.router.Post("/webhooks/stripe", h.StripeWebhook)
}

func (suite *PaymentHandlerTestSuite) createIntent(orderID string, authenticated bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID+"/payment", nil)
	if authenticated {
		token := jwtauth.New("HS256", []byte("secret"), nil)
		jwt, _, _ := token.Encode(map[string]interface{}{"jti": suite.userID.String()})
		req = req.WithContext(jwtauth.NewContext(context.Background(), jwt, nil))
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *PaymentHandlerTestSuite) TestCreatePaymentIntent_Success() {
	resp := &response.PaymentIntentResponse{ClientSecret: "pi_123_secret", Amount: 4028, Currency: "brl"}
	suite.service.On("CreatePaymentIntent", suite.userID, suite.orderID).Return(resp, nil)

	w := suite.createIntent(suite.orderID, true)

	suite.Equal(http.StatusOK, w.Code)
	var body response.PaymentIntentResponse
	suite.NoError(json.NewDecoder(w.Body).Decode(&body))
	suite.Equal("pi_123_secret", body.ClientSecret)
}

func (suite *PaymentHandlerTestSuite) TestCreatePaymentIntent_Unauthenticated() {
	w := suite.createIntent(suite.orderID, false)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *PaymentHandlerTestSuite) TestCreatePaymentIntent_InvalidID() {
	w := suite.createIntent("invalid", true)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *PaymentHandlerTestSuite) TestCreatePaymentIntent_ErrorMapping() {
	cases := map[string]struct {
		err    error
		status int
	}{
		"not found":    {service.ErrOrderNotFound, http.StatusNotFound},
		"not payable":  {service.ErrOrderNotPayable, http.StatusConflict},
		"stripe error": {fmt.Errorf("%w: boom", service.ErrPaymentProvider), http.StatusBadGateway},
		"unexpected":   {assert.AnError, http.StatusInternalServerError},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			suite.SetupTest()
			suite.service.On("CreatePaymentIntent", suite.userID, suite.orderID).Return(nil, tc.err)

			w := suite.createIntent(suite.orderID, true)

			suite.Equal(tc.status, w.Code)
		})
	}
}

func (suite *PaymentHandlerTestSuite) TestCreatePaymentIntent_HidesProviderErrorDetails() {
	suite.service.On("CreatePaymentIntent", suite.userID, suite.orderID).Return(nil, fmt.Errorf("%w: sk_live_secret", service.ErrPaymentProvider))

	w := suite.createIntent(suite.orderID, true)

	suite.NotContains(w.Body.String(), "sk_live_secret")
}

func (suite *PaymentHandlerTestSuite) webhook(body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(body))
	req.Header.Set("Stripe-Signature", "t=1,v1=abc")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *PaymentHandlerTestSuite) TestStripeWebhook_Success() {
	suite.service.On("HandleWebhook", []byte(`{"id":"evt_1"}`), "t=1,v1=abc").Return(nil)

	w := suite.webhook(`{"id":"evt_1"}`)

	suite.Equal(http.StatusOK, w.Code)
	suite.service.AssertExpectations(suite.T())
}

func (suite *PaymentHandlerTestSuite) TestStripeWebhook_InvalidSignature() {
	suite.service.On("HandleWebhook", mock.Anything, mock.Anything).Return(fmt.Errorf("%w: bad", service.ErrInvalidWebhookSignature))

	w := suite.webhook(`{}`)

	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *PaymentHandlerTestSuite) TestStripeWebhook_ProcessingErrorAsksForRetry() {
	suite.service.On("HandleWebhook", mock.Anything, mock.Anything).Return(assert.AnError)

	w := suite.webhook(`{}`)

	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *PaymentHandlerTestSuite) TestStripeWebhook_BodyTooLarge() {
	w := suite.webhook(strings.Repeat("a", 65*1024))

	suite.Equal(http.StatusBadRequest, w.Code)
	suite.service.AssertNotCalled(suite.T(), "HandleWebhook", mock.Anything, mock.Anything)
}

func TestPaymentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentHandlerTestSuite))
}
