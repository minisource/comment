package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/minisource/comment/config"
	"github.com/minisource/comment/internal/database"
	"github.com/minisource/comment/internal/handler"
	"github.com/minisource/comment/internal/middleware"
	"github.com/minisource/comment/internal/repository"
	"github.com/minisource/comment/internal/usecase"
	"github.com/minisource/go-common/logging"
	"github.com/minisource/go-sdk/auth"
)

// Router holds all dependencies for routing
type Router struct {
	app             *fiber.App
	cfg             *config.Config
	db              *database.MongoDB
	logger          logging.Logger
	commentHandler  *handler.CommentHandler
	reactionHandler *handler.ReactionHandler
	adminHandler    *handler.AdminHandler
	healthHandler   *handler.HealthHandler
}

// NewRouter creates a new router
func NewRouter(cfg *config.Config, db *database.MongoDB, logger logging.Logger) *Router {
	// Create repositories
	commentRepo := repository.NewCommentRepository(db)
	reactionRepo := repository.NewReactionRepository(db)
	reportRepo := repository.NewReportRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// Create notifier client (placeholder)
	var notifierClient usecase.NotifierClient = nil

	// Create usecases
	commentUsecase := usecase.NewCommentUsecase(commentRepo, reactionRepo, reportRepo, settingsRepo, notifierClient, cfg)
	reactionUsecase := usecase.NewReactionUsecase(commentRepo, reactionRepo)

	// Create handlers
	commentHandler := handler.NewCommentHandler(commentUsecase)
	reactionHandler := handler.NewReactionHandler(reactionUsecase)
	adminHandler := handler.NewAdminHandler(commentUsecase)
	healthHandler := handler.NewHealthHandler(db)

	return &Router{
		cfg:             cfg,
		db:              db,
		logger:          logger,
		commentHandler:  commentHandler,
		reactionHandler: reactionHandler,
		adminHandler:    adminHandler,
		healthHandler:   healthHandler,
	}
}

// Setup sets up the router
func (r *Router) Setup() *fiber.App {
	r.app = fiber.New(fiber.Config{
		ReadTimeout:  r.cfg.Server.ReadTimeout,
		WriteTimeout: r.cfg.Server.WriteTimeout,
		ErrorHandler: r.errorHandler,
	})

	// Global middleware
	r.app.Use(recover.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Tenant-ID",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	r.app.Use(middleware.LoggingMiddleware(r.logger))
	r.app.Use(middleware.TenantMiddleware())

	// Swagger route
	r.app.Get("/swagger/*", swagger.HandlerDefault)

	// Health routes (no auth required)
	r.app.Get("/health", r.healthHandler.HealthCheck)
	r.app.Get("/ready", r.healthHandler.Readiness)
	r.app.Get("/live", r.healthHandler.Liveness)

	// Setup auth middleware
	authClient := auth.NewClient(auth.ClientConfig{
		BaseURL: r.cfg.Auth.ServiceURL,
	})

	authMiddleware := middleware.AuthMiddleware(middleware.AuthConfig{
		AuthClient:   authClient,
		SkipPaths:    []string{"/health", "/ready", "/live"},
		RequireAdmin: []string{"/api/v1/admin"},
	})

	// API routes
	api := r.app.Group("/api/v1", authMiddleware)

	// Rate limiting for comment creation
	rateLimiter := middleware.RateLimitMiddleware(middleware.RateLimitConfig{
		Max:     r.cfg.Moderation.RateLimitPerMinute,
		Window:  time.Minute,
		KeyFunc: middleware.DefaultRateLimitKeyFunc,
	})

	// Comment routes
	comments := api.Group("/comments")
	comments.Post("/", rateLimiter, r.commentHandler.Create)
	comments.Get("/", r.commentHandler.List)
	comments.Get("/search", r.commentHandler.Search)
	comments.Get("/stats", r.commentHandler.GetStats)
	comments.Get("/:id", r.commentHandler.Get)
	comments.Put("/:id", r.commentHandler.Update)
	comments.Delete("/:id", r.commentHandler.Delete)
	comments.Get("/:id/replies", r.commentHandler.GetReplies)

	// Reaction routes
	comments.Post("/:id/reactions", r.reactionHandler.AddReaction)
	comments.Delete("/:id/reactions", r.reactionHandler.RemoveReaction)
	comments.Get("/:id/reactions/me", r.reactionHandler.GetUserReaction)

	// Admin routes
	admin := api.Group("/admin")
	adminComments := admin.Group("/comments")
	adminComments.Get("/pending", r.adminHandler.GetPendingComments)
	adminComments.Post("/:id/moderate", r.adminHandler.ModerateComment)
	adminComments.Post("/:id/pin", r.adminHandler.PinComment)
	adminComments.Delete("/:id", r.adminHandler.HardDelete)
	adminComments.Post("/bulk-moderate", r.adminHandler.BulkModerate)

	return r.app
}

// errorHandler handles errors
func (r *Router) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	r.logger.Error(logging.Internal, logging.Api, "Error handling request", map[logging.ExtraKey]interface{}{
		"error": err.Error(),
	})

	return c.Status(code).JSON(fiber.Map{
		"error":   "server_error",
		"message": message,
	})
}

// GetApp returns the fiber app
func (r *Router) GetApp() *fiber.App {
	return r.app
}
