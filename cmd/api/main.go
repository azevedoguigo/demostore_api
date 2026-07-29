package main

import (
	_ "github.com/azevedoguigo/demostore_api.git/docs"
	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/server"
)

// @title						Demostore API
// @version					1.0
// @description				API REST da Demostore, com gerenciamento de usuários, autenticação e produtos.
//
// @contact.name				Guilherme Azevedo
//
// @host						localhost:8080
// @BasePath					/api/v1
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Informe o token no formato: Bearer {token}
func main() {
	cfg := config.LoadConfig()
	server := server.NewServer(cfg)
	server.SetupServer()
	server.Start()
}
