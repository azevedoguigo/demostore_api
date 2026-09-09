package domain_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCategory_BindID(t *testing.T) {
	category := domain.Category{}

	assert.Equal(t, uuid.Nil, category.ID, "ID should be nil before binding")

	category.BindID()

	assert.NotNil(t, category.ID, "ID should be generated")
}
