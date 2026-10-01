package service_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) Create(cart *domain.Cart) error {
	return m.Called(cart).Error(0)
}

func (m *MockCartRepository) GetByUserID(userID uuid.UUID) (*domain.Cart, error) {
	args := m.Called(userID)
	c := args.Get(0)
	if c == nil {
		return nil, args.Error(1)
	}
	return c.(*domain.Cart), args.Error(1)
}

func (m *MockCartRepository) GetItem(cartID, productID uuid.UUID) (*domain.CartItem, error) {
	args := m.Called(cartID, productID)
	i := args.Get(0)
	if i == nil {
		return nil, args.Error(1)
	}
	return i.(*domain.CartItem), args.Error(1)
}

func (m *MockCartRepository) SaveItem(item *domain.CartItem) error {
	return m.Called(item).Error(0)
}

func (m *MockCartRepository) DeleteItem(cartID, productID uuid.UUID) error {
	return m.Called(cartID, productID).Error(0)
}

func (m *MockCartRepository) Clear(cartID uuid.UUID) error {
	return m.Called(cartID).Error(0)
}

type CartServiceTestSuite struct {
	suite.Suite
	repo        *MockCartRepository
	productRepo *MockProductRepository
	service     *service.CartServiceImpl
	userID      uuid.UUID
	cart        *domain.Cart
	product     *domain.Product
}

func (suite *CartServiceTestSuite) SetupTest() {
	suite.repo = new(MockCartRepository)
	suite.productRepo = new(MockProductRepository)
	suite.service = service.NewCartService(suite.repo, suite.productRepo)
	suite.userID = uuid.New()
	suite.cart = &domain.Cart{ID: uuid.New(), UserID: suite.userID}
	suite.product = &domain.Product{ID: uuid.New(), Name: "Product", Price: 10, Stock: 5}
}

func (suite *CartServiceTestSuite) TestGetCart_ExistingCart() {
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)

	cart, err := suite.service.GetCart(suite.userID)

	suite.NoError(err)
	suite.Equal(suite.cart, cart)
}

func (suite *CartServiceTestSuite) TestGetCart_CreatesWhenMissing() {
	suite.repo.On("GetByUserID", suite.userID).Return(nil, gorm.ErrRecordNotFound)
	suite.repo.On("Create", mock.MatchedBy(func(c *domain.Cart) bool {
		return c.UserID == suite.userID && c.ID != uuid.Nil
	})).Return(nil)

	cart, err := suite.service.GetCart(suite.userID)

	suite.NoError(err)
	suite.Equal(suite.userID, cart.UserID)
	suite.Empty(cart.Items)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CartServiceTestSuite) TestGetCart_CreateRaceReturnsExisting() {
	suite.repo.On("GetByUserID", suite.userID).Return(nil, gorm.ErrRecordNotFound).Once()
	suite.repo.On("Create", mock.AnythingOfType("*domain.Cart")).Return(assert.AnError)
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil).Once()

	cart, err := suite.service.GetCart(suite.userID)

	suite.NoError(err)
	suite.Equal(suite.cart, cart)
}

func (suite *CartServiceTestSuite) TestGetCart_RepositoryError() {
	suite.repo.On("GetByUserID", suite.userID).Return(nil, assert.AnError)

	_, err := suite.service.GetCart(suite.userID)

	suite.ErrorIs(err, assert.AnError)
}

func (suite *CartServiceTestSuite) TestAddItem_NewItem() {
	dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: 2}
	suite.productRepo.On("GetByID", suite.product.ID).Return(suite.product, nil)
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(nil, gorm.ErrRecordNotFound)
	suite.repo.On("SaveItem", mock.MatchedBy(func(i *domain.CartItem) bool {
		return i.Quantity == 2 && i.CartID == suite.cart.ID && i.ProductID == suite.product.ID && i.ID != uuid.Nil
	})).Return(nil)

	cart, err := suite.service.AddItem(suite.userID, dto)

	suite.NoError(err)
	suite.Equal(suite.cart, cart)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CartServiceTestSuite) TestAddItem_SumsExistingQuantity() {
	dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: 2}
	existing := &domain.CartItem{ID: uuid.New(), CartID: suite.cart.ID, ProductID: suite.product.ID, Quantity: 3}
	suite.productRepo.On("GetByID", suite.product.ID).Return(suite.product, nil)
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(existing, nil)
	suite.repo.On("SaveItem", existing).Return(nil)

	_, err := suite.service.AddItem(suite.userID, dto)

	suite.NoError(err)
	suite.Equal(5, existing.Quantity)
}

func (suite *CartServiceTestSuite) TestAddItem_InsufficientStockWithExistingQuantity() {
	dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: 3}
	existing := &domain.CartItem{ID: uuid.New(), CartID: suite.cart.ID, ProductID: suite.product.ID, Quantity: 3}
	suite.productRepo.On("GetByID", suite.product.ID).Return(suite.product, nil)
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(existing, nil)

	_, err := suite.service.AddItem(suite.userID, dto)

	suite.ErrorIs(err, service.ErrInsufficientStock)
	suite.repo.AssertNotCalled(suite.T(), "SaveItem", mock.Anything)
}

func (suite *CartServiceTestSuite) TestAddItem_InvalidQuantity() {
	for _, q := range []int{0, -1} {
		dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: q}

		_, err := suite.service.AddItem(suite.userID, dto)

		suite.ErrorIs(err, service.ErrInvalidQuantity)
	}
}

func (suite *CartServiceTestSuite) TestAddItem_InvalidProductID() {
	_, err := suite.service.AddItem(suite.userID, request.AddCartItemRequestDTO{ProductID: "invalid", Quantity: 1})

	suite.Error(err)
}

func (suite *CartServiceTestSuite) TestAddItem_ProductNotFound() {
	dto := request.AddCartItemRequestDTO{ProductID: suite.product.ID.String(), Quantity: 1}
	suite.productRepo.On("GetByID", suite.product.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.AddItem(suite.userID, dto)

	suite.ErrorIs(err, service.ErrProductNotFound)
}

func (suite *CartServiceTestSuite) TestUpdateItem_Success() {
	existing := &domain.CartItem{ID: uuid.New(), CartID: suite.cart.ID, ProductID: suite.product.ID, Quantity: 1}
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(existing, nil)
	suite.productRepo.On("GetByID", suite.product.ID).Return(suite.product, nil)
	suite.repo.On("SaveItem", existing).Return(nil)

	_, err := suite.service.UpdateItem(suite.userID, suite.product.ID.String(), request.UpdateCartItemRequestDTO{Quantity: 4})

	suite.NoError(err)
	suite.Equal(4, existing.Quantity)
}

func (suite *CartServiceTestSuite) TestUpdateItem_InsufficientStock() {
	existing := &domain.CartItem{ID: uuid.New(), CartID: suite.cart.ID, ProductID: suite.product.ID, Quantity: 1}
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(existing, nil)
	suite.productRepo.On("GetByID", suite.product.ID).Return(suite.product, nil)

	_, err := suite.service.UpdateItem(suite.userID, suite.product.ID.String(), request.UpdateCartItemRequestDTO{Quantity: 6})

	suite.ErrorIs(err, service.ErrInsufficientStock)
}

func (suite *CartServiceTestSuite) TestUpdateItem_InvalidQuantity() {
	_, err := suite.service.UpdateItem(suite.userID, suite.product.ID.String(), request.UpdateCartItemRequestDTO{Quantity: 0})

	suite.ErrorIs(err, service.ErrInvalidQuantity)
}

func (suite *CartServiceTestSuite) TestUpdateItem_ItemNotFound() {
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.UpdateItem(suite.userID, suite.product.ID.String(), request.UpdateCartItemRequestDTO{Quantity: 1})

	suite.ErrorIs(err, service.ErrCartItemNotFound)
}

func (suite *CartServiceTestSuite) TestUpdateItem_InvalidProductID() {
	_, err := suite.service.UpdateItem(suite.userID, "invalid", request.UpdateCartItemRequestDTO{Quantity: 1})

	suite.Error(err)
}

func (suite *CartServiceTestSuite) TestRemoveItem_Success() {
	existing := &domain.CartItem{ID: uuid.New(), CartID: suite.cart.ID, ProductID: suite.product.ID, Quantity: 1}
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(existing, nil)
	suite.repo.On("DeleteItem", suite.cart.ID, suite.product.ID).Return(nil)

	_, err := suite.service.RemoveItem(suite.userID, suite.product.ID.String())

	suite.NoError(err)
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CartServiceTestSuite) TestRemoveItem_ItemNotFound() {
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("GetItem", suite.cart.ID, suite.product.ID).Return(nil, gorm.ErrRecordNotFound)

	_, err := suite.service.RemoveItem(suite.userID, suite.product.ID.String())

	suite.ErrorIs(err, service.ErrCartItemNotFound)
}

func (suite *CartServiceTestSuite) TestRemoveItem_InvalidProductID() {
	_, err := suite.service.RemoveItem(suite.userID, "invalid")

	suite.Error(err)
}

func (suite *CartServiceTestSuite) TestClearCart_Success() {
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("Clear", suite.cart.ID).Return(nil)

	suite.NoError(suite.service.ClearCart(suite.userID))
	suite.repo.AssertExpectations(suite.T())
}

func (suite *CartServiceTestSuite) TestClearCart_RepositoryError() {
	suite.repo.On("GetByUserID", suite.userID).Return(suite.cart, nil)
	suite.repo.On("Clear", suite.cart.ID).Return(assert.AnError)

	suite.ErrorIs(suite.service.ClearCart(suite.userID), assert.AnError)
}

func TestCartServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CartServiceTestSuite))
}
