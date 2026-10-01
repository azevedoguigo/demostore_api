package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/database"
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/payment"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"gorm.io/gorm"
)

const shutdownTimeout = 10 * time.Second

type Server struct {
	router     *Router
	config     *config.Config
	db         *gorm.DB
	httpServer *http.Server
	orders     orderExpirer
	jobs       *backgroundJobs
}

func NewServer(cfg *config.Config) *Server {
	r := NewRouter()

	return &Server{
		router: r,
		config: cfg,
	}
}

func (s *Server) SetupServer() {
	db, err := database.NewPostgresDB(s.config)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	var userRepo domain.UserRepository = repository.NewUserRepository(db)
	var productRepo domain.ProductRepository = repository.NewProductRepository(db)
	var categoryRepo domain.CategoryRepository = repository.NewCategoryRepository(db)

	var cartRepo domain.CartRepository = repository.NewCartRepository(db)
	var orderRepo domain.OrderRepository = repository.NewOrderRepository(db)
	var paymentRepo domain.PaymentRepository = repository.NewPaymentRepository(db)
	var refundRepo domain.RefundRepository = repository.NewRefundRepository(db)

	if s.config.Stripe.SecretKey == "" || s.config.Stripe.WebhookSecret == "" {
		log.Println("WARNING: STRIPE_SECRET_KEY or STRIPE_WEBHOOK_SECRET is not set; payments will not work")
	}
	var paymentGateway domain.PaymentGateway = payment.NewStripeGateway(
		s.config.Stripe.SecretKey,
		s.config.Stripe.WebhookSecret,
		s.config.Stripe.BoletoExpiresAfterDays,
	)

	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userService)
	productService := service.NewProductService(productRepo, categoryRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	cartService := service.NewCartService(cartRepo, productRepo)
	paymentService := service.NewPaymentService(paymentRepo, orderRepo, refundRepo, paymentGateway)
	orderService := service.NewOrderService(orderRepo, cartRepo, paymentService, s.config.Order.PendingTTL)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	productHandler := handler.NewProductHandler(productService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	cartHandler := handler.NewCartHandler(cartService)
	orderHandler := handler.NewOrderHandler(orderService)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	s.router.SetupRoutes(
		userHandler,
		authHandler,
		productHandler,
		categoryHandler,
		cartHandler,
		orderHandler,
		paymentHandler,
	)

	s.db = db
	s.orders = orderService
}

func (s *Server) Start() {
	s.httpServer = &http.Server{
		Addr:    ":" + s.config.Postgres.ServerPort,
		Handler: s.router.chiRouter,
	}

	s.jobs = startBackgroundJobs(s.orders)

	go func() {
		log.Printf("Server starting on port %s", s.config.Postgres.ServerPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	s.waitForShutdown()
}

func (s *Server) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if s.jobs != nil {
		s.jobs.Stop()
	}

	if s.db != nil {
		if sqlDB, err := s.db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}
	}

	log.Println("Server exited gracefully")
}
