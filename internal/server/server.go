package server

import (
	"log"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
)

type Server struct {
	router *Router
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	r := NewRouter()

	return &Server{
		router: r,
		config: cfg,
	}
}

func (s *Server) SetupServer() {
	db, err := repository.NewPostgresDB(s.config)
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
}

func (s *Server) Start() {
	log.Printf("Server starting on port %s", s.config.Postgres.ServerPort)
	if err := http.ListenAndServe(":"+s.config.Postgres.ServerPort, s.router.chiRouter); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
