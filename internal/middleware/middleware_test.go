package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azevedoguigo/demostore_api.git/internal/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/stretchr/testify/suite"
)

var testTokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)

type MiddlewareTestSuite struct {
	suite.Suite
}

func (suite *MiddlewareTestSuite) newNextHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (suite *MiddlewareTestSuite) requestWithRoleClaim(role string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/products", nil)

	claims := map[string]interface{}{}
	if role != "" {
		claims["role"] = role
	}

	token, _, err := testTokenAuth.Encode(claims)
	suite.Require().NoError(err)

	ctx := jwtauth.NewContext(req.Context(), token, nil)

	return req.WithContext(ctx)
}

func (suite *MiddlewareTestSuite) TestRequireRole_AllowsMatchingRole() {
	req := suite.requestWithRoleClaim("admin")
	recorder := httptest.NewRecorder()

	handler := middleware.RequireRole("admin")(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusOK, recorder.Result().StatusCode)
}

func (suite *MiddlewareTestSuite) TestRequireRole_RejectsWrongRole() {
	req := suite.requestWithRoleClaim("customer")
	recorder := httptest.NewRecorder()

	handler := middleware.RequireRole("admin")(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusForbidden, recorder.Result().StatusCode)
}

func (suite *MiddlewareTestSuite) TestRequireRole_RejectsMissingRoleClaim() {
	req := suite.requestWithRoleClaim("")
	recorder := httptest.NewRecorder()

	handler := middleware.RequireRole("admin")(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusForbidden, recorder.Result().StatusCode)
}

func (suite *MiddlewareTestSuite) TestRequireRole_RejectsMissingToken() {
	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	recorder := httptest.NewRecorder()

	handler := middleware.RequireRole("admin")(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusForbidden, recorder.Result().StatusCode)
}

func (suite *MiddlewareTestSuite) TestAuthMiddleware_MissingHeader() {
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	recorder := httptest.NewRecorder()

	handler := middleware.AuthMiddleware(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusUnauthorized, recorder.Result().StatusCode)
}

func (suite *MiddlewareTestSuite) TestAuthMiddleware_NonBearerHeader() {
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Token abc123")
	recorder := httptest.NewRecorder()

	handler := middleware.AuthMiddleware(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusUnauthorized, recorder.Result().StatusCode)
}

func (suite *MiddlewareTestSuite) TestAuthMiddleware_InvalidToken() {
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	recorder := httptest.NewRecorder()

	handler := middleware.AuthMiddleware(suite.newNextHandler())
	handler.ServeHTTP(recorder, req)

	suite.Equal(http.StatusUnauthorized, recorder.Result().StatusCode)
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
