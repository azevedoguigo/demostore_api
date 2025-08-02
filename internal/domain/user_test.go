package domain_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser_BindID(t *testing.T) {
	user := domain.User{}

	assert.Equal(t, uuid.Nil, user.ID, "ID should be nil before binding")

	user.BindID()

	assert.NotNil(t, user.ID, "ID should be generated")
}
