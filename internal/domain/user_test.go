package domain_test

import (
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/dgrijalva/jwt-go"
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

func parseTestClaims(t *testing.T, tokenString string) *domain.UserClaims {
	claims := &domain.UserClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("dev-secret-change-me"), nil
	})
	assert.NoError(t, err, "Parsing generated token should not return an error")

	return claims
}

func TestUser_BindAccessToken_IncludesAdminRoleClaim(t *testing.T) {
	user := domain.User{Role: domain.RoleAdmin}
	err := user.BindAccessToken()

	assert.NoError(t, err)

	claims := parseTestClaims(t, user.AccessToken)
	assert.Equal(t, domain.RoleAdmin, claims.Role)
}

func TestUser_BindAccessToken_IncludesCustomerRoleClaim(t *testing.T) {
	user := domain.User{Role: domain.RoleCustomer}
	err := user.BindAccessToken()

	assert.NoError(t, err)

	claims := parseTestClaims(t, user.AccessToken)
	assert.Equal(t, domain.RoleCustomer, claims.Role)
}

func TestUser_IsAdmin(t *testing.T) {
	admin := domain.User{Role: domain.RoleAdmin}
	customer := domain.User{Role: domain.RoleCustomer}

	assert.True(t, admin.IsAdmin())
	assert.False(t, customer.IsAdmin())
}
