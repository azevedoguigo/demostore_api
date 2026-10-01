package service

import (
	"errors"
	"log"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEmptyCart               = errors.New("cart is empty")
	ErrOrderAmountOutOfRange   = errors.New("order total must be between R$ 0,50 and R$ 999.999,99")
	ErrOrderNotFound           = errors.New("order not found")
	ErrInvalidOrderStatus      = errors.New("invalid order status")
	ErrInvalidStatusTransition = domain.ErrInvalidStatusTransition
)

type OrderService interface {
	Checkout(userID uuid.UUID) (*domain.Order, error)
	GetUserOrders(userID uuid.UUID) ([]domain.Order, error)
	GetOrder(userID uuid.UUID, isAdmin bool, id string) (*domain.Order, error)
	CancelOrder(userID uuid.UUID, id string) (*domain.Order, error)
	GetAllOrders() ([]domain.Order, error)
	UpdateOrderStatus(id string, dto request.UpdateOrderStatusRequestDTO) (*domain.Order, error)
	ExpirePendingOrders(now time.Time) (int, error)
}

const (
	expirationBatchSize  = 100
	expirationRetryDelay = time.Hour
)

type OrderPaymentCanceler interface {
	CancelForOrder(orderID uuid.UUID) error
}

type OrderServiceImpl struct {
	repo       domain.OrderRepository
	cartRepo   domain.CartRepository
	payments   OrderPaymentCanceler
	pendingTTL time.Duration
}

func NewOrderService(
	repo domain.OrderRepository,
	cartRepo domain.CartRepository,
	payments OrderPaymentCanceler,
	pendingTTL time.Duration,
) *OrderServiceImpl {
	return &OrderServiceImpl{repo: repo, cartRepo: cartRepo, payments: payments, pendingTTL: pendingTTL}
}

func (s *OrderServiceImpl) Checkout(userID uuid.UUID) (*domain.Order, error) {
	cart, err := s.cartRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmptyCart
		}

		return nil, err
	}

	if len(cart.Items) == 0 {
		return nil, ErrEmptyCart
	}

	order := &domain.Order{
		UserID:   userID,
		Status:   domain.OrderStatusPending,
		Currency: domain.DefaultCurrency,
		Items:    make([]domain.OrderItem, 0, len(cart.Items)),
	}
	order.BindID()

	for _, cartItem := range cart.Items {
		if cartItem.Product == nil {
			return nil, ErrProductNotFound
		}

		item := domain.OrderItem{
			OrderID:     order.ID,
			ProductID:   cartItem.ProductID,
			ProductName: cartItem.Product.Name,
			UnitPrice:   domain.ToCents(cartItem.Product.Price),
			Quantity:    cartItem.Quantity,
		}
		item.BindID()
		item.Subtotal = item.UnitPrice * int64(item.Quantity)

		order.TotalAmount += item.Subtotal
		order.Items = append(order.Items, item)
	}

	if order.TotalAmount < domain.MinOrderAmount || order.TotalAmount > domain.MaxOrderAmount {
		return nil, ErrOrderAmountOutOfRange
	}

	expiresAt := time.Now().Add(s.pendingTTL)
	order.ExpiresAt = &expiresAt

	if err := s.repo.CreateFromCart(order, cart.ID); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderServiceImpl) GetUserOrders(userID uuid.UUID) ([]domain.Order, error) {
	return s.repo.GetByUserID(userID)
}

func (s *OrderServiceImpl) GetAllOrders() ([]domain.Order, error) {
	return s.repo.GetAll()
}

func (s *OrderServiceImpl) findOrder(id string) (*domain.Order, error) {
	orderID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	order, err := s.repo.GetByID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}

		return nil, err
	}

	return order, nil
}

func (s *OrderServiceImpl) GetOrder(userID uuid.UUID, isAdmin bool, id string) (*domain.Order, error) {
	order, err := s.findOrder(id)
	if err != nil {
		return nil, err
	}

	if !isAdmin && order.UserID != userID {
		return nil, ErrOrderNotFound
	}

	return order, nil
}

func (s *OrderServiceImpl) CancelOrder(userID uuid.UUID, id string) (*domain.Order, error) {
	order, err := s.GetOrder(userID, false, id)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.OrderStatusPending {
		return nil, ErrInvalidStatusTransition
	}

	return s.transition(order, domain.OrderStatusCancelled)
}

func (s *OrderServiceImpl) UpdateOrderStatus(id string, dto request.UpdateOrderStatusRequestDTO) (*domain.Order, error) {
	next := domain.OrderStatus(dto.Status)
	if !next.IsValid() {
		return nil, ErrInvalidOrderStatus
	}

	order, err := s.findOrder(id)
	if err != nil {
		return nil, err
	}

	return s.transition(order, next)
}

func (s *OrderServiceImpl) transition(order *domain.Order, next domain.OrderStatus) (*domain.Order, error) {
	if !order.CanTransitionTo(next) {
		return nil, ErrInvalidStatusTransition
	}

	if next == domain.OrderStatusCancelled {
		if err := s.payments.CancelForOrder(order.ID); err != nil {
			return nil, err
		}
	}

	from := order.Status
	order.Status = next

	if err := s.repo.UpdateStatus(order, from, next == domain.OrderStatusCancelled); err != nil {
		return nil, err
	}

	return order, nil
}

// ExpirePendingOrders cancels a batch of pending orders past their expiration, releasing their stock
// and payment intents. Failures are logged and retried on the next run.
func (s *OrderServiceImpl) ExpirePendingOrders(now time.Time) (int, error) {
	orders, err := s.repo.GetExpiredPending(now, expirationBatchSize)
	if err != nil {
		return 0, err
	}

	expired := 0
	for i := range orders {
		order := &orders[i]

		if _, err := s.transition(order, domain.OrderStatusCancelled); err != nil {
			log.Printf("order expiration: order %s not cancelled: %v", order.ID, err)

			if errors.Is(err, ErrPaymentInProgress) {
				if err := s.repo.ExtendExpiration(order.ID, now.Add(expirationRetryDelay)); err != nil {
					log.Printf("order expiration: order %s not postponed: %v", order.ID, err)
				}
			}
			continue
		}

		expired++
	}

	return expired, nil
}
