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

func (r *Router) setupCategoryRoutes(categoryHandler *handler.CategoryHandler) {
	r.chiRouter.Route("/api/v1/categories", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.With(middleware.RequireRole(domain.RoleAdmin)).Post("/", categoryHandler.CreateCategory)
		r.Get("/", categoryHandler.GetAllCategories)
		r.Get("/{id}", categoryHandler.GetCategoryByID)
		r.With(middleware.RequireRole(domain.RoleAdmin)).Put("/{id}", categoryHandler.UpdateCategory)
		r.With(middleware.RequireRole(domain.RoleAdmin)).Delete("/{id}", categoryHandler.DeleteCategory)
	})
}

func (r *Router) setupCartRoutes(cartHandler *handler.CartHandler) {
	r.chiRouter.Route("/api/v1/cart", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.Get("/", cartHandler.GetCart)
		r.Delete("/", cartHandler.ClearCart)
		r.Post("/items", cartHandler.AddItem)
		r.Put("/items/{product_id}", cartHandler.UpdateItem)
		r.Delete("/items/{product_id}", cartHandler.RemoveItem)
	})
}

func (r *Router) setupOrderRoutes(orderHandler *handler.OrderHandler) {
	r.chiRouter.Route("/api/v1/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.Post("/", orderHandler.Checkout)
		r.Get("/", orderHandler.GetMyOrders)
		r.Get("/{id}", orderHandler.GetOrder)
		r.Post("/{id}/cancel", orderHandler.CancelOrder)
	})

	r.chiRouter.Route("/api/v1/admin/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Use(middleware.RequireRole(domain.RoleAdmin))

		r.Get("/", orderHandler.GetAllOrders)
		r.Patch("/{id}/status", orderHandler.UpdateOrderStatus)
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
	categoryHandler *handler.CategoryHandler,
	cartHandler *handler.CartHandler,
	orderHandler *handler.OrderHandler,
) {
	r.setupUserRoutes(userHandler)
	r.setupAuthRoutes(authHandler)
	r.setupProductRoutes(productHandler)
	r.setupCategoryRoutes(categoryHandler)
	r.setupCartRoutes(cartHandler)
	r.setupOrderRoutes(orderHandler)
	r.setupSwaggerRoutes()
}
