package main

import (
	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/server"
)

func main() {
	cfg := config.LoadConfig()
	server := server.NewServer(cfg)
	server.SetupServer()
	server.Start()
}
