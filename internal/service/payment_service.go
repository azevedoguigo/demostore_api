package service

import (
	"errors"
	"log"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrderNotPayable         = errors.New("order is not awaiting payment")
	ErrPaymentAlreadyProcessed = errors.New("payment was already processed by the provider, try again in a moment")
	ErrPaymentProvider         = domain.ErrPaymentProvider
	ErrInvalidWebhookSignature = domain.ErrInvalidWebhookSignature
)

type PaymentService interface {
	CreatePaymentIntent(userID uuid.UUID, orderID string) (*response.PaymentIntentResponse, error)
	HandleWebhook(payload []byte, signature string) error
	CancelForOrder(orderID uuid.UUID) error
}

type PaymentServiceImpl struct {
	repo      domain.PaymentRepository
	orderRepo domain.OrderRepository
	gateway   domain.PaymentGateway
}

func NewPaymentService(repo domain.PaymentRepository, orderRepo domain.OrderRepository, gateway domain.PaymentGateway) *PaymentServiceImpl {
	return &PaymentServiceImpl{repo: repo, orderRepo: orderRepo, gateway: gateway}
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

// CreatePaymentIntent starts (or resumes) the payment of a pending order, returning the
// client secret the frontend needs to confirm the payment with Stripe.
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

	pi, err := s.gateway.CreatePaymentIntent(order.ID, order.TotalAmount, order.Currency)
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
		// A concurrent request already stored it; the idempotency key guarantees it's the same intent.
		existing, getErr := s.repo.GetByOrderID(order.ID)
		if getErr != nil {
			return nil, err
		}

		payment = existing
	}

	return newPaymentIntentResponse(payment, pi), nil
}

// CancelForOrder releases the order's payment before the order is cancelled: an open intent is
// cancelled at the provider and a captured payment is refunded. Orders without payment are a no-op.
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

	case domain.PaymentStatusSucceeded:
		if err := s.gateway.RefundPaymentIntent(payment.ProviderPaymentID); err != nil {
			return err
		}
		payment.Status = domain.PaymentStatusRefunded

	default:
		if err := s.gateway.CancelPaymentIntent(payment.ProviderPaymentID); err != nil {
			if !errors.Is(err, domain.ErrPaymentIntentUnexpectedState) {
				return err
			}

			// Already cancelled at Stripe is fine; anything else (succeeded, processing) means the
			// customer just paid and the webhook hasn't arrived yet, so the order must not be cancelled.
			pi, getErr := s.gateway.GetPaymentIntent(payment.ProviderPaymentID)
			if getErr != nil {
				return getErr
			}
			if pi.Status != "canceled" {
				return ErrPaymentAlreadyProcessed
			}
		}
		payment.Status = domain.PaymentStatusCancelled
	}

	return s.repo.Update(payment)
}

// HandleWebhook applies a verified provider event. Every branch is idempotent, since the
// provider retries deliveries and may send them out of order.
func (s *PaymentServiceImpl) HandleWebhook(payload []byte, signature string) error {
	event, err := s.gateway.ParseWebhookEvent(payload, signature)
	if err != nil {
		return err
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
	}

	return nil
}

func (s *PaymentServiceImpl) handlePaymentSucceeded(payment *domain.Payment) error {
	if payment.Status == domain.PaymentStatusSucceeded || payment.Status == domain.PaymentStatusRefunded {
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
			// The order changed concurrently (most likely cancelled); re-read to decide below.
			if order, err = s.orderRepo.GetByID(payment.OrderID); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// The customer paid an order that was cancelled in the meantime: give the money back.
	if order.Status == domain.OrderStatusCancelled {
		if err := s.gateway.RefundPaymentIntent(payment.ProviderPaymentID); err != nil {
			return err
		}

		payment.Status = domain.PaymentStatusRefunded
		return s.repo.Update(payment)
	}

	payment.Status = domain.PaymentStatusSucceeded
	return s.repo.Update(payment)
}

// handlePaymentFailed keeps the order pending: Stripe lets the customer retry with the same intent.
func (s *PaymentServiceImpl) handlePaymentFailed(payment *domain.Payment) error {
	if payment.Status != domain.PaymentStatusPending {
		return nil
	}

	payment.Status = domain.PaymentStatusFailed
	return s.repo.Update(payment)
}

func (s *PaymentServiceImpl) handlePaymentCanceled(payment *domain.Payment) error {
	if payment.Status == domain.PaymentStatusSucceeded || payment.Status == domain.PaymentStatusRefunded {
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
