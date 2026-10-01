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

type CartRepositoryTestSuite struct {
	suite.Suite
	db      *gorm.DB
	repo    *repository.CartRepository
	product *domain.Product
}

func (s *CartRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Cart{}, &domain.CartItem{})
	s.repo = repository.NewCartRepository(db)

	category := &domain.Category{Name: "Category", Description: "Category description"}
	category.BindID()
	db.Create(category)

	s.product = &domain.Product{Name: "Product", Description: "Product description", Price: 10, Stock: 5, CategoryID: category.ID}
	s.product.BindID()
	db.Create(s.product)
}

func (s *CartRepositoryTestSuite) newCart() *domain.Cart {
	cart := &domain.Cart{UserID: uuid.New()}
	cart.BindID()
	assert.Nil(s.T(), s.repo.Create(cart))
	return cart
}

func (s *CartRepositoryTestSuite) newItem(cart *domain.Cart, quantity int) *domain.CartItem {
	item := &domain.CartItem{CartID: cart.ID, ProductID: s.product.ID, Quantity: quantity}
	item.BindID()
	assert.Nil(s.T(), s.repo.SaveItem(item))
	return item
}

func (s *CartRepositoryTestSuite) TestCreate_DuplicateUser() {
	cart := s.newCart()

	other := &domain.Cart{UserID: cart.UserID}
	other.BindID()

	assert.NotNil(s.T(), s.repo.Create(other), "a user can only have one cart")
}

func (s *CartRepositoryTestSuite) TestGetByUserID_PreloadsItemsAndProducts() {
	cart := s.newCart()
	s.newItem(cart, 2)

	found, err := s.repo.GetByUserID(cart.UserID)

	assert.Nil(s.T(), err)
	assert.Len(s.T(), found.Items, 1)
	assert.Equal(s.T(), s.product.Name, found.Items[0].Product.Name)
}

func (s *CartRepositoryTestSuite) TestGetByUserID_NotFound() {
	_, err := s.repo.GetByUserID(uuid.New())

	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}

func (s *CartRepositoryTestSuite) TestGetItem_Success() {
	cart := s.newCart()
	item := s.newItem(cart, 3)

	found, err := s.repo.GetItem(cart.ID, s.product.ID)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), item.ID, found.ID)
	assert.Equal(s.T(), 3, found.Quantity)
}

func (s *CartRepositoryTestSuite) TestGetItem_NotFound() {
	cart := s.newCart()

	_, err := s.repo.GetItem(cart.ID, s.product.ID)

	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}

func (s *CartRepositoryTestSuite) TestSaveItem_UpdatesQuantity() {
	cart := s.newCart()
	item := s.newItem(cart, 1)

	item.Quantity = 4
	assert.Nil(s.T(), s.repo.SaveItem(item))

	found, _ := s.repo.GetItem(cart.ID, s.product.ID)
	assert.Equal(s.T(), 4, found.Quantity)
}

func (s *CartRepositoryTestSuite) TestSaveItem_DuplicateProductInCart() {
	cart := s.newCart()
	s.newItem(cart, 1)

	dup := &domain.CartItem{CartID: cart.ID, ProductID: s.product.ID, Quantity: 1}
	dup.BindID()

	assert.NotNil(s.T(), s.repo.SaveItem(dup))
}

func (s *CartRepositoryTestSuite) TestDeleteItem_AllowsReAdding() {
	cart := s.newCart()
	s.newItem(cart, 1)

	assert.Nil(s.T(), s.repo.DeleteItem(cart.ID, s.product.ID))
	_, err := s.repo.GetItem(cart.ID, s.product.ID)
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)

	s.newItem(cart, 1)
}

func (s *CartRepositoryTestSuite) TestClear_OnlyAffectsGivenCart() {
	cart := s.newCart()
	other := s.newCart()
	s.newItem(cart, 1)
	s.newItem(other, 1)

	assert.Nil(s.T(), s.repo.Clear(cart.ID))

	_, err := s.repo.GetItem(cart.ID, s.product.ID)
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
	_, err = s.repo.GetItem(other.ID, s.product.ID)
	assert.Nil(s.T(), err)
}

func TestCartRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(CartRepositoryTestSuite))
}
