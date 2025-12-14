package main

import (
	"log"
	"os"

	"developer-portal-backend/internal/api/routes"
	"developer-portal-backend/internal/config"
	"developer-portal-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	_ "developer-portal-backend/docs" // This is needed for swag
)

//	@title			Developer Portal Backend API
//	@version		1.0
//	@description	This is the backend API for the Developer Portal, providing endpoints for managing organizations, teams, projects, components, and deployments.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.example.com/support
//	@contact.email	support@example.com

//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT

//	@BasePath	/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.

func main() {
	// Load environment variables from .env file in development
	if err := godotenv.Load(); err != nil {
		logrus.Info("No .env file found, using system environment variables")
	}

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Set up logging
	setupLogging(cfg.LogLevel)

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL, nil)
	if err != nil {
		logrus.Fatal("Failed to initialize database:", err)
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := routes.SetupRoutes(db, cfg)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "7008"
	}

	logrus.Infof("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		logrus.Fatal("Failed to start server:", err)
	}
}

func setupLogging(level string) {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	switch level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}
