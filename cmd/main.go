package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/minisource/comment/config"
	_ "github.com/minisource/comment/docs" // Swagger docs
	"github.com/minisource/comment/internal/database"
	"github.com/minisource/comment/internal/router"
	"github.com/minisource/go-common/logging"
)

// @title Comment Service API
// @version 1.0
// @description Comment management service for Minisource
// @host localhost:5010
// @BasePath /api/v1
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logging.NewLogger(&logging.LoggerConfig{
		FilePath: "logs/comment.log",
		Encoding: "json",
		Level:    cfg.Logging.Level,
		Logger:   "zap",
	})

	logger.Info(logging.General, logging.Startup, "Starting comment service...", nil)

	// Initialize MongoDB
	db, err := database.NewMongoDB(cfg.MongoDB)
	if err != nil {
		logger.Fatal(logging.General, logging.Startup, "Failed to connect to MongoDB", map[logging.ExtraKey]interface{}{
			"error": err.Error(),
		})
	}
	defer func() {
		if err := db.Close(context.Background()); err != nil {
			logger.Error(logging.General, logging.Startup, "Failed to close MongoDB", map[logging.ExtraKey]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Create indexes
	if err := db.CreateIndexes(context.Background()); err != nil {
		logger.Error(logging.General, logging.Startup, "Failed to create indexes", map[logging.ExtraKey]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info(logging.General, logging.Startup, "MongoDB connected successfully", nil)

	// Setup router
	r := router.NewRouter(cfg, db, logger)
	app := r.Setup()

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		logger.Info(logging.General, logging.Startup, fmt.Sprintf("Server starting on %s", addr), nil)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(logging.General, logging.Startup, "Shutting down server...", nil)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error(logging.General, logging.Startup, "Server forced to shutdown", map[logging.ExtraKey]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info(logging.General, logging.Startup, "Server exited", nil)
}
