package service

import (
	"errors"
	"log"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrderNotPayable         = errors.New("order is not awaiting payment")
	ErrOrderExpired            = errors.New("order payment window has expired")
	ErrPaymentInProgress       = errors.New("payment is being processed by the provider and can't be cancelled now")
	ErrPaymentProvider         = domain.ErrPaymentProvider
	ErrInvalidWebhookSignature = domain.ErrInvalidWebhookSignature
)

type PaymentService interface {
	CreatePaymentIntent(userID uuid.UUID, orderID string) (*response.PaymentIntentResponse, error)
	HandleWebhook(payload []byte, signature string) error
	CancelForOrder(orderID uuid.UUID) error
	RefundItems(orderID string, dto request.CreateRefundRequestDTO) (*domain.Refund, error)
	GetOrderRefunds(orderID string) ([]domain.Refund, error)
}

type PaymentServiceImpl struct {
	repo       domain.PaymentRepository
	orderRepo  domain.OrderRepository
	refundRepo domain.RefundRepository
	gateway    domain.PaymentGateway
	now        func() time.Time
}

func NewPaymentService(repo domain.PaymentRepository, orderRepo domain.OrderRepository, refundRepo domain.RefundRepository, gateway domain.PaymentGateway) *PaymentServiceImpl {
	return &PaymentServiceImpl{repo: repo, orderRepo: orderRepo, refundRepo: refundRepo, gateway: gateway, now: time.Now}
}

func newPaymentIntentResponse(payment *domain.Payment, pi *domain.PaymentIntent) *response.PaymentIntentResponse {
	return &response.PaymentIntentResponse{
		PaymentID:    payment.ID,
		OrderID:      payment.OrderID,
		ClientSecret: pi.ClientSecret,
		Amount:       payment.Amount,
		Currency:     payment.Currency,
		Status:       pi.Status,
	}
}

func (s *PaymentServiceImpl) CreatePaymentIntent(userID uuid.UUID, orderID string) (*response.PaymentIntentResponse, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, err
	}

	order, err := s.orderRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}

		return nil, err
	}

	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}

	if order.Status != domain.OrderStatusPending {
		return nil, ErrOrderNotPayable
	}

	if order.ExpiresAt != nil && !s.now().Before(*order.ExpiresAt) {
		return nil, ErrOrderExpired
	}

	payment, err := s.repo.GetByOrderID(order.ID)
	if err == nil {
		pi, err := s.gateway.GetPaymentIntent(payment.ProviderPaymentID)
		if err != nil {
			return nil, err
		}

		return newPaymentIntentResponse(payment, pi), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	req := domain.PaymentIntentRequest{
		OrderID:            order.ID,
		Amount:             order.TotalAmount,
		Currency:           order.Currency,
		PaymentMethodTypes: domain.PaymentMethodTypesFor(order.TotalAmount),
	}
	if order.ExpiresAt != nil {
		req.ExpiresAt = *order.ExpiresAt
	}

	pi, err := s.gateway.CreatePaymentIntent(req)
	if err != nil {
		return nil, err
	}

	payment = &domain.Payment{
		OrderID:           order.ID,
		Provider:          domain.PaymentProviderStripe,
		ProviderPaymentID: pi.ID,
		Status:            domain.PaymentStatusPending,
		Amount:            order.TotalAmount,
		Currency:          order.Currency,
	}
	payment.BindID()

	if err := s.repo.Create(payment); err != nil {
		existing, getErr := s.repo.GetByOrderID(order.ID)
		if getErr != nil {
			return nil, err
		}

		payment = existing
	}

	return newPaymentIntentResponse(payment, pi), nil
}

func (s *PaymentServiceImpl) CancelForOrder(orderID uuid.UUID) error {
	payment, err := s.repo.GetByOrderID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	switch payment.Status {
	case domain.PaymentStatusCancelled, domain.PaymentStatusRefunded:
		return nil

	case domain.PaymentStatusSucceeded, domain.PaymentStatusPartiallyRefunded:
		return s.refundRemaining(payment, "order cancelled")
	}

	if err := s.gateway.CancelPaymentIntent(payment.ProviderPaymentID); err != nil {
		if !errors.Is(err, domain.ErrPaymentIntentUnexpectedState) {
			return err
		}

		pi, getErr := s.gateway.GetPaymentIntent(payment.ProviderPaymentID)
		if getErr != nil {
			return getErr
		}
		if pi.Status != domain.PaymentIntentStatusCanceled {
			return ErrPaymentInProgress
		}
	}

	payment.Status = domain.PaymentStatusCancelled
	return s.repo.Update(payment)
}

func (s *PaymentServiceImpl) HandleWebhook(payload []byte, signature string) error {
	event, err := s.gateway.ParseWebhookEvent(payload, signature)
	if err != nil {
		return err
	}

	if event.Refund != nil {
		return s.handleRefundEvent(event.Refund)
	}

	if event.PaymentIntentID == "" {
		return nil
	}

	payment, err := s.repo.GetByProviderPaymentID(event.PaymentIntentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("stripe webhook: ignoring %s for unknown payment intent %s", event.Type, event.PaymentIntentID)
			return nil
		}

		return err
	}

	switch event.Type {
	case domain.PaymentEventSucceeded:
		return s.handlePaymentSucceeded(payment)
	case domain.PaymentEventFailed:
		return s.handlePaymentFailed(payment)
	case domain.PaymentEventCanceled:
		return s.handlePaymentCanceled(payment)
	case domain.PaymentEventRequiresAction:
		return s.handlePaymentRequiresAction(payment, event)
	}

	return nil
}

func (s *PaymentServiceImpl) handlePaymentSucceeded(payment *domain.Payment) error {
	if payment.Status == domain.PaymentStatusRefunded || payment.Status == domain.PaymentStatusPartiallyRefunded {
		return nil
	}

	order, err := s.orderRepo.GetByID(payment.OrderID)
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusPending {
		order.Status = domain.OrderStatusPaid

		err := s.orderRepo.UpdateStatus(order, domain.OrderStatusPending, false)
		if errors.Is(err, domain.ErrInvalidStatusTransition) {
			if order, err = s.orderRepo.GetByID(payment.OrderID); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	if payment.Status != domain.PaymentStatusSucceeded {
		payment.Status = domain.PaymentStatusSucceeded
		if err := s.repo.Update(payment); err != nil {
			return err
		}
	}

	if order.Status != domain.OrderStatusCancelled {
		return nil
	}

	err = s.refundRemaining(payment, "paid after the order was cancelled")
	switch {
	case errors.Is(err, ErrRefundInProgress):
		return nil
	case errors.Is(err, ErrRefundNotSupported):
		log.Printf("MANUAL REFUND REQUIRED: order %s was cancelled but its boleto payment %s succeeded", order.ID, payment.ProviderPaymentID)
		return nil
	}

	return err
}

func (s *PaymentServiceImpl) handlePaymentFailed(payment *domain.Payment) error {
	if payment.Status != domain.PaymentStatusPending {
		return nil
	}

	payment.Status = domain.PaymentStatusFailed
	return s.repo.Update(payment)
}

func (s *PaymentServiceImpl) handlePaymentCanceled(payment *domain.Payment) error {
	switch payment.Status {
	case domain.PaymentStatusSucceeded, domain.PaymentStatusRefunded, domain.PaymentStatusPartiallyRefunded:
		return nil
	}

	order, err := s.orderRepo.GetByID(payment.OrderID)
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusPending {
		order.Status = domain.OrderStatusCancelled

		err := s.orderRepo.UpdateStatus(order, domain.OrderStatusPending, true)
		if err != nil && !errors.Is(err, domain.ErrInvalidStatusTransition) {
			return err
		}
	}

	if payment.Status == domain.PaymentStatusCancelled {
		return nil
	}

	payment.Status = domain.PaymentStatusCancelled
	return s.repo.Update(payment)
}

func (s *PaymentServiceImpl) handlePaymentRequiresAction(payment *domain.Payment, event *domain.PaymentEvent) error {
	if event.NextActionType != domain.NextActionBoletoDisplayDetails || event.NextActionExpiresAt.IsZero() {
		return nil
	}

	return s.orderRepo.ExtendExpiration(payment.OrderID, event.NextActionExpiresAt.Add(domain.BoletoConfirmationGrace))
}
