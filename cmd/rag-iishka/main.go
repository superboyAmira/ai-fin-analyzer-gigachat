package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"rag-iishka/internal/api"
	"rag-iishka/internal/api/handlers"
	"rag-iishka/internal/repository"
	"rag-iishka/internal/service"
	"rag-iishka/pkg/auth"
	"rag-iishka/pkg/config"
	"rag-iishka/pkg/logger"
	"rag-iishka/pkg/postgres"

	"go.uber.org/zap"
)

// @title RAG Iishka API
// @version 1.0
// @description RAG-сервис анализа расходов по изображениям финансовых документов
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@rag-iishka.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize global logger
	if err := logger.Init(cfg.Logger.Level); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	appLogger := logger.Get()
	appLogger.Info("Starting RAG Iishka service")

	// Initialize database
	ctx := context.Background()
	db, err := postgres.NewPool(ctx, &cfg.Database, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db, appLogger)
	docRepo := repository.NewDocumentRepository(db, appLogger)
	txRepo := repository.NewTransactionRepository(db, appLogger)
	recRepo := repository.NewRecommendationRepository(db, appLogger)
	knowledgeRepo := repository.NewKnowledgeRepository(db, appLogger)

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Expiration, cfg.JWT.RefreshExp)

	// Initialize services
	authService := service.NewAuthService(userRepo, jwtManager, appLogger)

	llmService, err := service.NewLLMService(&cfg.GigaChat, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize LLM service", zap.Error(err))
	}
	defer llmService.Close()

	ocrService := service.NewOCRService(llmService, appLogger)

	ragService := service.NewRAGService(knowledgeRepo, &cfg.RAG, appLogger)
	recService := service.NewRecommendationService(llmService, ragService, recRepo, appLogger)

	uploadDir := "uploads"
	docService := service.NewDocumentService(docRepo, txRepo, recRepo, ocrService, llmService, recService, uploadDir, appLogger)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, appLogger)
	docHandler := handlers.NewDocumentHandler(docService, appLogger)

	// Setup router
	app := api.SetupRouter(authHandler, docHandler, jwtManager, appLogger)

	// Start server
	go func() {
		addr := ":" + cfg.Server.Port
		appLogger.Info("Server starting", zap.String("address", addr))
		if err := app.Listen(addr); err != nil {
			appLogger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server")
	if err := app.Shutdown(); err != nil {
		appLogger.Error("Server shutdown error", zap.Error(err))
	}
}
