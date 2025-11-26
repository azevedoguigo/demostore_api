package server

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/middleware"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
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

		r.Post("/", productHandler.CreateProduct)
		r.Get("/", productHandler.GetAllProducts)
		r.Get("/{id}", productHandler.GetProductByID)
	})
}

func (r *Router) SetupRoutes(db *gorm.DB) {
	var userRepo domain.UserRepository = repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	authService := service.NewAuthService(userService)
	authHandler := handler.NewAuthHandler(authService)

	var productRepo domain.ProductRepository = repository.NewProductRepository(db)
	productService := service.NewProductService(productRepo)
	productHandler := handler.NewProductHandler(productService)

	r.setupUserRoutes(userHandler)
	r.setupAuthRoutes(authHandler)
	r.setupProductRoutes(productHandler)
}
