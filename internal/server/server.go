package server

import (
	"log"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/handler"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	return &Server{
		router: r,
		config: cfg,
	}
}

func (s *Server) SetupRoutes() {
	db, err := repository.NewPostgresDB(s.config)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	var userRepo domain.UserRepository = repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	s.router.Route("/api/v1/users", func(r chi.Router) {
		r.Post("/", userHandler.CreateUser)
		r.Get("/{id}", userHandler.GetUserByID)
	})

	authService := service.NewAuthService(userService)
	authHandler := handler.NewAuthHandler(authService)

	s.router.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
	})
}

func (s *Server) Start() {
	log.Printf("Server starting on port %s", s.config.ServerPort)
	if err := http.ListenAndServe(":"+s.config.ServerPort, s.router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
