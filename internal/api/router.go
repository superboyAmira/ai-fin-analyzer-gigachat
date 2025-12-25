package api

import (
	"os"
	"path/filepath"
	"rag-iishka/docs"
	"rag-iishka/internal/api/handlers"
	"rag-iishka/pkg/auth"
	"rag-iishka/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"go.uber.org/zap"
)

func SetupRouter(
	authHandler *handlers.AuthHandler,
	docHandler *handlers.DocumentHandler,
	jwtManager *auth.JWTManager,
	appLogger *zap.Logger,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
	app.Use(logger.New())

	// Swagger - импорт docs пакета регистрирует документацию через init()
	_ = docs.SwaggerInfo // ensure docs package is imported and init() is called
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Get project root directory
	// Try to find web/static directory relative to current working directory
	// or relative to executable location
	webStaticPath := findWebStaticPath(appLogger)
	uploadsPath := findUploadsPath(appLogger)

	// Static files (web interface)
	if webStaticPath != "" {
		appLogger.Info("Serving static files", zap.String("path", webStaticPath))
		app.Static("/static", webStaticPath)
	} else {
		appLogger.Warn("Web static directory not found, static files will not be served")
	}
	if uploadsPath != "" {
		appLogger.Info("Serving uploads", zap.String("path", uploadsPath))
		app.Static("/uploads", uploadsPath)
	}

	// Serve index.html for root path
	app.Get("/", func(c *fiber.Ctx) error {
		indexPath := filepath.Join(webStaticPath, "index.html")
		if webStaticPath == "" || !fileExists(indexPath) {
			// Fallback: try common paths
			paths := []string{
				"./web/static/index.html",
				"web/static/index.html",
				"../web/static/index.html",
				"../../web/static/index.html",
			}
			for _, path := range paths {
				if fileExists(path) {
					return c.SendFile(path)
				}
			}
			return c.Status(404).SendString("Web interface not found. Please ensure web/static/index.html exists.")
		}
		return c.SendFile(indexPath)
	})

	// API routes
	api := app.Group("/user")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)

	// Protected routes
	protected := app.Group("/api/v1", middleware.AuthMiddleware(jwtManager, appLogger))

	// Document routes
	documents := protected.Group("/documents")
	documents.Post("/upload", docHandler.UploadDocument)
	documents.Get("", docHandler.ListDocuments)
	documents.Post("/:id/process", docHandler.ProcessDocument)

	return app
}

// findWebStaticPath finds the path to web/static directory
func findWebStaticPath(logger *zap.Logger) string {
	// Get current working directory
	cwd, _ := os.Getwd()
	logger.Info("Current working directory", zap.String("cwd", cwd))

	// Try paths relative to current working directory
	paths := []string{
		"./web/static",
		"web/static",
		"../web/static",
		"../../web/static",
	}

	for _, path := range paths {
		fullPath := filepath.Join(cwd, path)
		if fileExists(filepath.Join(path, "index.html")) {
			logger.Info("Found web static path", zap.String("path", path), zap.String("full", fullPath))
			return path
		}
		logger.Debug("Tried path", zap.String("path", path), zap.String("full", fullPath), zap.Bool("exists", fileExists(filepath.Join(path, "index.html"))))
	}

	return ""
}

// findUploadsPath finds the path to uploads directory
func findUploadsPath(logger *zap.Logger) string {
	// Try paths relative to current working directory
	paths := []string{
		"./uploads",
		"uploads",
		"../uploads",
		"../../uploads",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "uploads" // Default, will be created if needed
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
