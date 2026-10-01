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

type RefundRepositoryTestSuite struct {
	suite.Suite
	db      *gorm.DB
	repo    *repository.RefundRepository
	product *domain.Product
	order   *domain.Order
	payment *domain.Payment
}

func (s *RefundRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Order{}, &domain.OrderItem{},
		&domain.Payment{}, &domain.Refund{}, &domain.RefundItem{})
	s.repo = repository.NewRefundRepository(db)

	category := &domain.Category{Name: "Category", Description: "Category description"}
	category.BindID()
	db.Create(category)

	s.product = &domain.Product{Name: "Product", Description: "Product description", Price: 10, Stock: 7, CategoryID: category.ID}
	s.product.BindID()
	s.Require().NoError(db.Create(s.product).Error)

	s.order = &domain.Order{UserID: uuid.New(), Status: domain.OrderStatusPaid, Currency: "brl", TotalAmount: 3000}
	s.order.BindID()
	item := domain.OrderItem{OrderID: s.order.ID, ProductID: s.product.ID, ProductName: "Product", UnitPrice: 1000, Quantity: 3, Subtotal: 3000}
	item.BindID()
	s.order.Items = []domain.OrderItem{item}
	s.Require().NoError(db.Create(s.order).Error)

	s.payment = &domain.Payment{OrderID: s.order.ID, Provider: "stripe", ProviderPaymentID: "pi_1", Status: domain.PaymentStatusSucceeded, Amount: 3000, Currency: "brl"}
	s.payment.BindID()
	s.Require().NoError(db.Create(s.payment).Error)
}

func (s *RefundRepositoryTestSuite) newRefund(quantity int, restock bool) *domain.Refund {
	refund := &domain.Refund{
		OrderID:   s.order.ID,
		PaymentID: s.payment.ID,
		Amount:    int64(quantity) * 1000,
		Status:    domain.RefundStatusPending,
		Restock:   restock,
	}
	refund.BindID()

	if quantity > 0 {
		item := domain.RefundItem{RefundID: refund.ID, OrderItemID: s.order.Items[0].ID, ProductID: s.product.ID, Quantity: quantity, Amount: refund.Amount}
		item.BindID()
		refund.Items = []domain.RefundItem{item}
	}

	return refund
}

func (s *RefundRepositoryTestSuite) refundedQuantity() int {
	var item domain.OrderItem
	s.Require().NoError(s.db.First(&item, "id = ?", s.order.Items[0].ID).Error)
	return item.RefundedQuantity
}

func (s *RefundRepositoryTestSuite) stock() int {
	var product domain.Product
	s.Require().NoError(s.db.Unscoped().First(&product, "id = ?", s.product.ID).Error)
	return product.Stock
}

func (s *RefundRepositoryTestSuite) storedPayment() domain.Payment {
	var payment domain.Payment
	s.Require().NoError(s.db.First(&payment, "id = ?", s.payment.ID).Error)
	return payment
}

func (s *RefundRepositoryTestSuite) TestReserve_ReservesQuantities() {
	refund := s.newRefund(2, true)

	assert.Nil(s.T(), s.repo.Reserve(refund))

	assert.Equal(s.T(), 2, s.refundedQuantity())
	found, err := s.repo.GetByID(refund.ID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), domain.RefundStatusPending, found.Status)
	assert.Len(s.T(), found.Items, 1)
}

func (s *RefundRepositoryTestSuite) TestReserve_CantExceedPurchasedQuantity() {
	s.Require().NoError(s.repo.Reserve(s.newRefund(2, true)))

	err := s.repo.Reserve(s.newRefund(2, true))

	assert.ErrorIs(s.T(), err, domain.ErrRefundQuantityExceeded)
	assert.Equal(s.T(), 2, s.refundedQuantity(), "a rejected reservation must not change anything")

	refunds, _ := s.repo.GetByOrderID(s.order.ID)
	assert.Len(s.T(), refunds, 1)
}

func (s *RefundRepositoryTestSuite) TestReserve_ItemFromAnotherOrder() {
	refund := s.newRefund(1, true)
	refund.OrderID = uuid.New()

	assert.ErrorIs(s.T(), s.repo.Reserve(refund), domain.ErrRefundQuantityExceeded)
}

func (s *RefundRepositoryTestSuite) TestMarkSucceeded_PartialRefundRestocks() {
	refund := s.newRefund(1, true)
	s.Require().NoError(s.repo.Reserve(refund))

	settled, err := s.repo.MarkSucceeded(refund)

	assert.Nil(s.T(), err)
	assert.True(s.T(), settled)
	assert.Equal(s.T(), domain.RefundStatusSucceeded, refund.Status)
	assert.Equal(s.T(), 8, s.stock())
	payment := s.storedPayment()
	assert.Equal(s.T(), int64(1000), payment.RefundedAmount)
	assert.Equal(s.T(), domain.PaymentStatusPartiallyRefunded, payment.Status)
}

func (s *RefundRepositoryTestSuite) TestMarkSucceeded_FullRefundWithoutRestock() {
	refund := s.newRefund(0, false)
	refund.Amount = 3000
	s.Require().NoError(s.repo.Reserve(refund))

	_, err := s.repo.MarkSucceeded(refund)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 7, s.stock())
	payment := s.storedPayment()
	assert.Equal(s.T(), int64(3000), payment.RefundedAmount)
	assert.Equal(s.T(), domain.PaymentStatusRefunded, payment.Status)
}

func (s *RefundRepositoryTestSuite) TestMarkSucceeded_AppliesOnlyOnce() {
	refund := s.newRefund(1, true)
	s.Require().NoError(s.repo.Reserve(refund))
	s.Require().NoError(func() error { _, err := s.repo.MarkSucceeded(refund); return err }())

	// A duplicated webhook delivery.
	stale := s.newRefund(1, true)
	stale.ID = refund.ID
	settled, err := s.repo.MarkSucceeded(stale)

	assert.Nil(s.T(), err)
	assert.False(s.T(), settled)
	assert.Equal(s.T(), 8, s.stock())
	assert.Equal(s.T(), int64(1000), s.storedPayment().RefundedAmount)
}

func (s *RefundRepositoryTestSuite) TestMarkFailed_ReleasesReservation() {
	refund := s.newRefund(2, true)
	s.Require().NoError(s.repo.Reserve(refund))

	settled, err := s.repo.MarkFailed(refund)

	assert.Nil(s.T(), err)
	assert.True(s.T(), settled)
	assert.Equal(s.T(), 0, s.refundedQuantity())
	assert.Equal(s.T(), 7, s.stock())
	assert.Equal(s.T(), int64(0), s.storedPayment().RefundedAmount)

	// The released units can be refunded again.
	assert.Nil(s.T(), s.repo.Reserve(s.newRefund(3, true)))
}

func (s *RefundRepositoryTestSuite) TestMarkFailed_AfterSuccessIsIgnored() {
	refund := s.newRefund(1, true)
	s.Require().NoError(s.repo.Reserve(refund))
	_, _ = s.repo.MarkSucceeded(refund)

	settled, err := s.repo.MarkFailed(refund)

	assert.Nil(s.T(), err)
	assert.False(s.T(), settled)
	assert.Equal(s.T(), 1, s.refundedQuantity())
}

func (s *RefundRepositoryTestSuite) TestProviderIDAndQueries() {
	sent := s.newRefund(1, true)
	unsent := s.newRefund(1, true)
	s.Require().NoError(s.repo.Reserve(sent))
	s.Require().NoError(s.repo.Reserve(unsent))
	s.Require().NoError(s.repo.SetProviderRefundID(sent.ID, "re_1"))

	found, err := s.repo.GetByProviderRefundID("re_1")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), sent.ID, found.ID)

	pending, err := s.repo.GetPending(s.payment.ID)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), pending, 2)

	_, _ = s.repo.MarkSucceeded(sent)
	pending, _ = s.repo.GetPending(s.payment.ID)
	assert.Len(s.T(), pending, 1)
	assert.Equal(s.T(), unsent.ID, pending[0].ID)
	assert.Nil(s.T(), pending[0].ProviderRefundID)

	_, err = s.repo.GetByProviderRefundID("re_missing")
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}

func TestRefundRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RefundRepositoryTestSuite))
}
