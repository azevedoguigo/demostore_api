package server

import (
	"log"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/repository"
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

	s.router.SetupRoutes(db)
}

func (s *Server) Start() {
	log.Printf("Server starting on port %s", s.config.ServerPort)
	if err := http.ListenAndServe(":"+s.config.ServerPort, s.router.chiRouter); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
