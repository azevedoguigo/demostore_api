package server

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/middleware"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Router struct {
	chiRouter *chi.Mux
}

func NewRouter() *Router {
	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	return &Router{
		chiRouter: r,
	}
}

func (r *Router) setupUserRoutes(userHandler *handler.UserHandler) {
	r.chiRouter.Route("/api/v1/users", func(r chi.Router) {
		r.Post("/", userHandler.CreateUser)
		r.With(middleware.AuthMiddleware).Get("/{id}", userHandler.GetUserByID)
	})
}

func (r *Router) setupAuthRoutes(authHandler *handler.AuthHandler) {
	r.chiRouter.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
	})
}

func (s *Router) setupProductRoutes(productHandler *handler.ProductHandler) {
	s.chiRouter.Route("/api/v1/products", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.With(middleware.RequireRole(domain.RoleAdmin)).Post("/", productHandler.CreateProduct)
		r.Get("/", productHandler.GetAllProducts)
		r.Get("/{id}", productHandler.GetProductByID)
		r.With(middleware.RequireRole(domain.RoleAdmin)).Put("/{id}", productHandler.UpdateProduct)
		r.With(middleware.RequireRole(domain.RoleAdmin)).Delete("/{id}", productHandler.DeleteProduct)
	})
}

func (r *Router) setupSwaggerRoutes() {
	r.chiRouter.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
}

func (r *Router) SetupRoutes(
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	productHandler *handler.ProductHandler,
) {
	r.setupUserRoutes(userHandler)
	r.setupAuthRoutes(authHandler)
	r.setupProductRoutes(productHandler)
	r.setupSwaggerRoutes()
}
