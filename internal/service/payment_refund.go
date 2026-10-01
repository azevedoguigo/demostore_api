package service

import (
	"errors"
	"log"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrderNotRefundable     = errors.New("order has no captured payment to refund")
	ErrInvalidRefundItems     = errors.New("refund must list products of the order with quantities greater than zero")
	ErrRefundQuantityExceeded = domain.ErrRefundQuantityExceeded
	ErrRefundInProgress       = errors.New("order has a refund in progress, try again once it completes")
	ErrRefundNotSupported     = errors.New("boleto payments can't be refunded through Stripe; refund the customer outside of it")
)

func (s *PaymentServiceImpl) RefundItems(orderID string, dto request.CreateRefundRequestDTO) (*domain.Refund, error) {
	order, err := s.findOrder(orderID)
	if err != nil {
		return nil, err
	}

	switch order.Status {
	case domain.OrderStatusPaid, domain.OrderStatusShipped, domain.OrderStatusDelivered:
	default:
		return nil, ErrOrderNotRefundable
	}

	payment, err := s.repo.GetByOrderID(order.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotRefundable
		}

		return nil, err
	}

	if payment.Status != domain.PaymentStatusSucceeded && payment.Status != domain.PaymentStatusPartiallyRefunded {
		return nil, ErrOrderNotRefundable
	}

	refund, err := buildItemRefund(order, payment, dto)
	if err != nil {
		return nil, err
	}

	if err := s.ensureRefundable(payment); err != nil {
		return nil, err
	}

	return s.createRefund(payment, refund)
}

func (s *PaymentServiceImpl) GetOrderRefunds(orderID string) ([]domain.Refund, error) {
	order, err := s.findOrder(orderID)
	if err != nil {
		return nil, err
	}

	return s.refundRepo.GetByOrderID(order.ID)
}

func (s *PaymentServiceImpl) findOrder(orderID string) (*domain.Order, error) {
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

	return order, nil
}

func buildItemRefund(order *domain.Order, payment *domain.Payment, dto request.CreateRefundRequestDTO) (*domain.Refund, error) {
	if len(dto.Items) == 0 {
		return nil, ErrInvalidRefundItems
	}

	itemsByProduct := make(map[uuid.UUID]*domain.OrderItem, len(order.Items))
	for i := range order.Items {
		itemsByProduct[order.Items[i].ProductID] = &order.Items[i]
	}

	quantities := make(map[uuid.UUID]int)
	var productOrder []uuid.UUID
	for _, requested := range dto.Items {
		productID, err := uuid.Parse(requested.ProductID)
		if err != nil || requested.Quantity < 1 || itemsByProduct[productID] == nil {
			return nil, ErrInvalidRefundItems
		}

		if _, seen := quantities[productID]; !seen {
			productOrder = append(productOrder, productID)
		}
		quantities[productID] += requested.Quantity
	}

	refund := &domain.Refund{
		OrderID:   order.ID,
		PaymentID: payment.ID,
		Reason:    dto.Reason,
		Status:    domain.RefundStatusPending,
		Restock:   true,
	}
	refund.BindID()

	for _, productID := range productOrder {
		orderItem := itemsByProduct[productID]
		quantity := quantities[productID]

		if quantity > orderItem.Quantity-orderItem.RefundedQuantity {
			return nil, ErrRefundQuantityExceeded
		}

		item := domain.RefundItem{
			RefundID:    refund.ID,
			OrderItemID: orderItem.ID,
			ProductID:   productID,
			Quantity:    quantity,
			Amount:      orderItem.UnitPrice * int64(quantity),
		}
		item.BindID()

		refund.Amount += item.Amount
		refund.Items = append(refund.Items, item)
	}

	return refund, nil
}

func (s *PaymentServiceImpl) ensureRefundable(payment *domain.Payment) error {
	method, err := s.gateway.GetPaymentMethodType(payment.ProviderPaymentID)
	if err != nil {
		return err
	}

	if !domain.RefundSupported(method) {
		return ErrRefundNotSupported
	}

	return nil
}

func (s *PaymentServiceImpl) refundRemaining(payment *domain.Payment, reason string) error {
	pending, resent, err := s.resendUnsentRefunds(payment)
	if err != nil {
		return err
	}
	if pending {
		return ErrRefundInProgress
	}

	if resent {
		if payment, err = s.repo.GetByOrderID(payment.OrderID); err != nil {
			return err
		}
	}

	amount := payment.RemainingRefundable()
	if amount <= 0 {
		return nil
	}

	if err := s.ensureRefundable(payment); err != nil {
		return err
	}

	refund := &domain.Refund{
		OrderID:   payment.OrderID,
		PaymentID: payment.ID,
		Amount:    amount,
		Reason:    reason,
		Status:    domain.RefundStatusPending,
	}
	refund.BindID()

	_, err = s.createRefund(payment, refund)
	return err
}

func (s *PaymentServiceImpl) resendUnsentRefunds(payment *domain.Payment) (stillPending, resent bool, err error) {
	pending, err := s.refundRepo.GetPending(payment.ID)
	if err != nil {
		return false, false, err
	}

	for i := range pending {
		refund := &pending[i]
		if refund.ProviderRefundID == nil {
			resent = true
			if err := s.sendRefund(payment, refund); err != nil {
				return false, resent, err
			}
		}

		if refund.Status == domain.RefundStatusPending {
			stillPending = true
		}
	}

	return stillPending, resent, nil
}

func (s *PaymentServiceImpl) createRefund(payment *domain.Payment, refund *domain.Refund) (*domain.Refund, error) {
	if err := s.refundRepo.Reserve(refund); err != nil {
		return nil, err
	}

	if err := s.sendRefund(payment, refund); err != nil {
		return nil, err
	}

	return refund, nil
}

func (s *PaymentServiceImpl) sendRefund(payment *domain.Payment, refund *domain.Refund) error {
	providerRefund, err := s.gateway.RefundPaymentIntent(payment.ProviderPaymentID, refund.Amount, refund.ID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentProviderRejected) {
			if _, markErr := s.refundRepo.MarkFailed(refund); markErr != nil {
				log.Printf("refund %s: failed to release reservation: %v", refund.ID, markErr)
			}
		}

		return err
	}

	if err := s.refundRepo.SetProviderRefundID(refund.ID, providerRefund.ID); err != nil {
		return err
	}
	refund.ProviderRefundID = &providerRefund.ID

	return s.applyRefundStatus(refund, providerRefund.Status)
}

func (s *PaymentServiceImpl) applyRefundStatus(refund *domain.Refund, providerStatus string) error {
	switch providerStatus {
	case domain.ProviderRefundSucceeded:
		_, err := s.refundRepo.MarkSucceeded(refund)
		return err

	case domain.ProviderRefundFailed, domain.ProviderRefundCanceled:
		settled, err := s.refundRepo.MarkFailed(refund)
		if settled {
			log.Printf("refund %s for order %s %s at the provider; the customer was not refunded", refund.ID, refund.OrderID, providerStatus)
		}
		return err
	}

	// pending / requires_action: wait for the next refund event.
	return nil
}

func (s *PaymentServiceImpl) handleRefundEvent(providerRefund *domain.ProviderRefund) error {
	refund, err := s.findRefund(providerRefund)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("stripe webhook: ignoring unknown refund %s (not created through the API)", providerRefund.ID)
			return nil
		}

		return err
	}

	if refund.ProviderRefundID == nil {
		if err := s.refundRepo.SetProviderRefundID(refund.ID, providerRefund.ID); err != nil {
			return err
		}
		refund.ProviderRefundID = &providerRefund.ID
	}

	return s.applyRefundStatus(refund, providerRefund.Status)
}

func (s *PaymentServiceImpl) findRefund(providerRefund *domain.ProviderRefund) (*domain.Refund, error) {
	if id, err := uuid.Parse(providerRefund.LocalRefundID); err == nil {
		return s.refundRepo.GetByID(id)
	}

	return s.refundRepo.GetByProviderRefundID(providerRefund.ID)
}
