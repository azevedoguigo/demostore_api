package repository_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type OrderRepositoryTestSuite struct {
	suite.Suite
	db       *gorm.DB
	repo     *repository.OrderRepository
	cartRepo *repository.CartRepository
	productA *domain.Product
	productB *domain.Product
	cart     *domain.Cart
}

func (s *OrderRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Cart{}, &domain.CartItem{}, &domain.Order{}, &domain.OrderItem{})
	s.repo = repository.NewOrderRepository(db)
	s.cartRepo = repository.NewCartRepository(db)

	category := &domain.Category{Name: "Category", Description: "Category description"}
	category.BindID()
	db.Create(category)

	s.productA = s.newProduct(category.ID, "Product A", 5)
	s.productB = s.newProduct(category.ID, "Product B", 2)

	s.cart = &domain.Cart{UserID: uuid.New()}
	s.cart.BindID()
	s.cartRepo.Create(s.cart)
	s.addToCart(s.productA, 3)
	s.addToCart(s.productB, 2)
}

func (s *OrderRepositoryTestSuite) newProduct(categoryID uuid.UUID, name string, stock int) *domain.Product {
	p := &domain.Product{Name: name, Description: "Product description", Price: 10, Stock: stock, CategoryID: categoryID}
	p.BindID()
	s.Require().NoError(s.db.Create(p).Error)
	return p
}

func (s *OrderRepositoryTestSuite) addToCart(p *domain.Product, quantity int) {
	item := &domain.CartItem{CartID: s.cart.ID, ProductID: p.ID, Quantity: quantity}
	item.BindID()
	s.Require().NoError(s.cartRepo.SaveItem(item))
}

func (s *OrderRepositoryTestSuite) newOrder(items map[*domain.Product]int) *domain.Order {
	order := &domain.Order{UserID: s.cart.UserID, Status: domain.OrderStatusPending, Currency: domain.DefaultCurrency}
	order.BindID()
	for p, q := range items {
		item := domain.OrderItem{OrderID: order.ID, ProductID: p.ID, ProductName: p.Name, UnitPrice: 1000, Quantity: q, Subtotal: int64(q) * 1000}
		item.BindID()
		order.TotalAmount += item.Subtotal
		order.Items = append(order.Items, item)
	}
	return order
}

func (s *OrderRepositoryTestSuite) stockOf(p *domain.Product) int {
	var found domain.Product
	s.Require().NoError(s.db.Unscoped().First(&found, "id = ?", p.ID).Error)
	return found.Stock
}

func (s *OrderRepositoryTestSuite) cartItemCount() int64 {
	var count int64
	s.db.Model(&domain.CartItem{}).Where("cart_id = ?", s.cart.ID).Count(&count)
	return count
}

func (s *OrderRepositoryTestSuite) TestCreateFromCart_Success() {
	order := s.newOrder(map[*domain.Product]int{s.productA: 3, s.productB: 2})

	err := s.repo.CreateFromCart(order, s.cart.ID)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 2, s.stockOf(s.productA))
	assert.Equal(s.T(), 0, s.stockOf(s.productB))
	assert.Equal(s.T(), int64(0), s.cartItemCount())

	found, err := s.repo.GetByID(order.ID)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), found.Items, 2)
	assert.Equal(s.T(), int64(5000), found.TotalAmount)
}

func (s *OrderRepositoryTestSuite) TestCreateFromCart_InsufficientStockRollsBack() {
	order := s.newOrder(map[*domain.Product]int{s.productA: 3, s.productB: 3})

	err := s.repo.CreateFromCart(order, s.cart.ID)

	assert.ErrorIs(s.T(), err, domain.ErrInsufficientStock)
	assert.Contains(s.T(), err.Error(), "Product B")
	assert.Equal(s.T(), 5, s.stockOf(s.productA), "stock of other items must be restored")
	assert.Equal(s.T(), 2, s.stockOf(s.productB))
	assert.Equal(s.T(), int64(2), s.cartItemCount(), "cart must be kept")

	_, err = s.repo.GetByID(order.ID)
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}

func (s *OrderRepositoryTestSuite) TestCreateFromCart_DeletedProduct() {
	s.db.Delete(s.productB)
	order := s.newOrder(map[*domain.Product]int{s.productB: 1})

	err := s.repo.CreateFromCart(order, s.cart.ID)

	assert.ErrorIs(s.T(), err, domain.ErrInsufficientStock)
}

func (s *OrderRepositoryTestSuite) TestGetByUserID_OnlyUserOrders() {
	mine := s.newOrder(map[*domain.Product]int{s.productA: 1})
	s.Require().NoError(s.repo.CreateFromCart(mine, s.cart.ID))

	other := s.newOrder(map[*domain.Product]int{s.productA: 1})
	other.UserID = uuid.New()
	s.Require().NoError(s.repo.CreateFromCart(other, uuid.New()))

	orders, err := s.repo.GetByUserID(s.cart.UserID)

	assert.Nil(s.T(), err)
	assert.Len(s.T(), orders, 1)
	assert.Equal(s.T(), mine.ID, orders[0].ID)
	assert.Len(s.T(), orders[0].Items, 1)

	all, err := s.repo.GetAll()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), all, 2)
}

func (s *OrderRepositoryTestSuite) TestUpdateStatus_Success() {
	order := s.newOrder(map[*domain.Product]int{s.productA: 3})
	s.Require().NoError(s.repo.CreateFromCart(order, s.cart.ID))

	order.Status = domain.OrderStatusPaid
	err := s.repo.UpdateStatus(order, domain.OrderStatusPending, false)

	assert.Nil(s.T(), err)
	found, _ := s.repo.GetByID(order.ID)
	assert.Equal(s.T(), domain.OrderStatusPaid, found.Status)
	assert.Equal(s.T(), 2, s.stockOf(s.productA))
}

func (s *OrderRepositoryTestSuite) TestUpdateStatus_CancelRestoresStock() {
	order := s.newOrder(map[*domain.Product]int{s.productA: 3, s.productB: 2})
	s.Require().NoError(s.repo.CreateFromCart(order, s.cart.ID))
	s.db.Delete(s.productB)

	order.Status = domain.OrderStatusCancelled
	err := s.repo.UpdateStatus(order, domain.OrderStatusPending, true)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 5, s.stockOf(s.productA))
	assert.Equal(s.T(), 2, s.stockOf(s.productB), "soft-deleted products also get stock back")
}

func (s *OrderRepositoryTestSuite) TestUpdateStatus_StaleStatusConflict() {
	order := s.newOrder(map[*domain.Product]int{s.productA: 3})
	s.Require().NoError(s.repo.CreateFromCart(order, s.cart.ID))

	order.Status = domain.OrderStatusCancelled
	s.Require().NoError(s.repo.UpdateStatus(order, domain.OrderStatusPending, true))

	// A second concurrent cancel still believes the order is pending.
	err := s.repo.UpdateStatus(order, domain.OrderStatusPending, true)

	assert.ErrorIs(s.T(), err, domain.ErrInvalidStatusTransition)
	assert.Equal(s.T(), 5, s.stockOf(s.productA), "stock must not be restored twice")
}

func TestOrderRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(OrderRepositoryTestSuite))
}
