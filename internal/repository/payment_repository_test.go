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

type PaymentRepositoryTestSuite struct {
	suite.Suite
	repo *repository.PaymentRepository
}

func (s *PaymentRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		s.T().Fatal(err)
	}
	db.AutoMigrate(&domain.Payment{})
	s.repo = repository.NewPaymentRepository(db)
}

func (s *PaymentRepositoryTestSuite) newPayment(orderID uuid.UUID, providerID string) *domain.Payment {
	p := &domain.Payment{
		OrderID:           orderID,
		Provider:          domain.PaymentProviderStripe,
		ProviderPaymentID: providerID,
		Status:            domain.PaymentStatusPending,
		Amount:            1000,
		Currency:          domain.DefaultCurrency,
	}
	p.BindID()
	return p
}

func (s *PaymentRepositoryTestSuite) TestCreateAndGet() {
	p := s.newPayment(uuid.New(), "pi_1")
	assert.Nil(s.T(), s.repo.Create(p))

	byOrder, err := s.repo.GetByOrderID(p.OrderID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), p.ID, byOrder.ID)

	byProvider, err := s.repo.GetByProviderPaymentID("pi_1")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), p.ID, byProvider.ID)
}

func (s *PaymentRepositoryTestSuite) TestCreate_OnePaymentPerOrder() {
	orderID := uuid.New()
	assert.Nil(s.T(), s.repo.Create(s.newPayment(orderID, "pi_1")))

	assert.NotNil(s.T(), s.repo.Create(s.newPayment(orderID, "pi_2")))
}

func (s *PaymentRepositoryTestSuite) TestCreate_UniqueProviderPaymentID() {
	assert.Nil(s.T(), s.repo.Create(s.newPayment(uuid.New(), "pi_1")))

	assert.NotNil(s.T(), s.repo.Create(s.newPayment(uuid.New(), "pi_1")))
}

func (s *PaymentRepositoryTestSuite) TestGet_NotFound() {
	_, err := s.repo.GetByOrderID(uuid.New())
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)

	_, err = s.repo.GetByProviderPaymentID("pi_missing")
	assert.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}

func (s *PaymentRepositoryTestSuite) TestUpdate() {
	p := s.newPayment(uuid.New(), "pi_1")
	s.Require().NoError(s.repo.Create(p))

	p.Status = domain.PaymentStatusSucceeded
	assert.Nil(s.T(), s.repo.Update(p))

	found, _ := s.repo.GetByOrderID(p.OrderID)
	assert.Equal(s.T(), domain.PaymentStatusSucceeded, found.Status)
}

func TestPaymentRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentRepositoryTestSuite))
}
