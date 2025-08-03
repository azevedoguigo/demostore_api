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

func TestUser_HashPassword(t *testing.T) {
	password := "securepassword"
	user := domain.User{}
	err := user.BindHashedPassword(password)

	assert.NoError(t, err, "Hashing password should not return an error")
}

func TestUser_BindAccessToken(t *testing.T) {
	user := domain.User{}
	err := user.BindAccessToken()

	assert.NoError(t, err, "Binding access token should not return an error")
	assert.NotEmpty(t, user.AccessToken, "Access token should be generated")
}
