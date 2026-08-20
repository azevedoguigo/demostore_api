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

	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userService)
	productService := service.NewProductService(productRepo)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	productHandler := handler.NewProductHandler(productService)

	s.router.SetupRoutes(
		userHandler,
		authHandler,
		productHandler,
	)

	s.db = db
}

func (s *Server) Start() {
	s.httpServer = &http.Server{
		Addr:    ":" + s.config.Postgres.ServerPort,
		Handler: s.router.chiRouter,
	}

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

	if s.db != nil {
		if sqlDB, err := s.db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}
	}

	log.Println("Server exited gracefully")
}
