package service

import (
	"errors"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
	ErrInsufficientStock = domain.ErrInsufficientStock
)

type CartService interface {
	GetCart(userID uuid.UUID) (*domain.Cart, error)
	AddItem(userID uuid.UUID, dto request.AddCartItemRequestDTO) (*domain.Cart, error)
	UpdateItem(userID uuid.UUID, productID string, dto request.UpdateCartItemRequestDTO) (*domain.Cart, error)
	RemoveItem(userID uuid.UUID, productID string) (*domain.Cart, error)
	ClearCart(userID uuid.UUID) error
}

type CartServiceImpl struct {
	repo        domain.CartRepository
	productRepo domain.ProductRepository
}

func NewCartService(repo domain.CartRepository, productRepo domain.ProductRepository) *CartServiceImpl {
	return &CartServiceImpl{repo: repo, productRepo: productRepo}
}

// getOrCreateCart returns the user's cart, creating an empty one on first use.
func (s *CartServiceImpl) getOrCreateCart(userID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.repo.GetByUserID(userID)
	if err == nil {
		return cart, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cart = &domain.Cart{UserID: userID}
	cart.BindID()

	if err := s.repo.Create(cart); err != nil {
		// A concurrent request may have created the cart first (unique user_id).
		if existing, getErr := s.repo.GetByUserID(userID); getErr == nil {
			return existing, nil
		}

		return nil, err
	}

	cart.Items = []domain.CartItem{}
	return cart, nil
}

func (s *CartServiceImpl) GetCart(userID uuid.UUID) (*domain.Cart, error) {
	return s.getOrCreateCart(userID)
}

func (s *CartServiceImpl) AddItem(userID uuid.UUID, dto request.AddCartItemRequestDTO) (*domain.Cart, error) {
	productID, err := uuid.Parse(dto.ProductID)
	if err != nil {
		return nil, err
	}

	if dto.Quantity < 1 {
		return nil, ErrInvalidQuantity
	}

	product, err := s.productRepo.GetByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}

		return nil, err
	}

	cart, err := s.getOrCreateCart(userID)
	if err != nil {
		return nil, err
	}

	item, err := s.repo.GetItem(cart.ID, productID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		item = &domain.CartItem{CartID: cart.ID, ProductID: productID}
		item.BindID()
	}

	item.Quantity += dto.Quantity
	if item.Quantity > product.Stock {
		return nil, ErrInsufficientStock
	}

	if err := s.repo.SaveItem(item); err != nil {
		return nil, err
	}

	return s.repo.GetByUserID(userID)
}

func (s *CartServiceImpl) UpdateItem(userID uuid.UUID, productID string, dto request.UpdateCartItemRequestDTO) (*domain.Cart, error) {
	pid, err := uuid.Parse(productID)
	if err != nil {
		return nil, err
	}

	if dto.Quantity < 1 {
		return nil, ErrInvalidQuantity
	}

	cart, err := s.getOrCreateCart(userID)
	if err != nil {
		return nil, err
	}

	item, err := s.repo.GetItem(cart.ID, pid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCartItemNotFound
		}

		return nil, err
	}

	product, err := s.productRepo.GetByID(pid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}

		return nil, err
	}

	if dto.Quantity > product.Stock {
		return nil, ErrInsufficientStock
	}

	item.Quantity = dto.Quantity
	if err := s.repo.SaveItem(item); err != nil {
		return nil, err
	}

	return s.repo.GetByUserID(userID)
}

func (s *CartServiceImpl) RemoveItem(userID uuid.UUID, productID string) (*domain.Cart, error) {
	pid, err := uuid.Parse(productID)
	if err != nil {
		return nil, err
	}

	cart, err := s.getOrCreateCart(userID)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.GetItem(cart.ID, pid); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCartItemNotFound
		}

		return nil, err
	}

	if err := s.repo.DeleteItem(cart.ID, pid); err != nil {
		return nil, err
	}

	return s.repo.GetByUserID(userID)
}

func (s *CartServiceImpl) ClearCart(userID uuid.UUID) error {
	cart, err := s.getOrCreateCart(userID)
	if err != nil {
		return err
	}

	return s.repo.Clear(cart.ID)
}
