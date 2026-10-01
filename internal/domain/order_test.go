package domain_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrder_BindID(t *testing.T) {
	order := domain.Order{}
	order.BindID()
	assert.NotEqual(t, uuid.Nil, order.ID)

	item := domain.OrderItem{}
	item.BindID()
	assert.NotEqual(t, uuid.Nil, item.ID)
}

func TestOrderStatus_IsValid(t *testing.T) {
	for _, s := range []domain.OrderStatus{"pending", "paid", "shipped", "delivered", "cancelled"} {
		assert.True(t, s.IsValid(), s)
	}

	assert.False(t, domain.OrderStatus("refunded").IsValid())
	assert.False(t, domain.OrderStatus("").IsValid())
}

func TestOrder_CanTransitionTo(t *testing.T) {
	cases := []struct {
		from, to domain.OrderStatus
		allowed  bool
	}{
		{domain.OrderStatusPending, domain.OrderStatusPaid, true},
		{domain.OrderStatusPending, domain.OrderStatusCancelled, true},
		{domain.OrderStatusPending, domain.OrderStatusShipped, false},
		{domain.OrderStatusPaid, domain.OrderStatusShipped, true},
		{domain.OrderStatusPaid, domain.OrderStatusCancelled, true},
		{domain.OrderStatusPaid, domain.OrderStatusPending, false},
		{domain.OrderStatusShipped, domain.OrderStatusDelivered, true},
		{domain.OrderStatusShipped, domain.OrderStatusCancelled, false},
		{domain.OrderStatusDelivered, domain.OrderStatusCancelled, false},
		{domain.OrderStatusCancelled, domain.OrderStatusPaid, false},
	}

	for _, tc := range cases {
		order := domain.Order{Status: tc.from}
		assert.Equal(t, tc.allowed, order.CanTransitionTo(tc.to), "%s -> %s", tc.from, tc.to)
	}
}

func TestToCents(t *testing.T) {
	assert.Equal(t, int64(1999), domain.ToCents(19.99))
	assert.Equal(t, int64(30), domain.ToCents(0.3))
	assert.Equal(t, int64(1000), domain.ToCents(10))
	assert.Equal(t, int64(29), domain.ToCents(0.29))
}
