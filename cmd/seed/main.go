package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rag-iishka/internal/models"
	"rag-iishka/internal/repository"
	"rag-iishka/internal/service"
	"rag-iishka/pkg/config"
	"rag-iishka/pkg/logger"
	"rag-iishka/pkg/postgres"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logger.Level); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	appLogger := logger.Get()

	// Connect to database
	ctx := context.Background()
	db, err := postgres.NewPool(ctx, &cfg.Database, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	knowledgeRepo := repository.NewKnowledgeRepository(db, appLogger)

	// Initialize LLM service for PDF processing
	llmService, err := service.NewLLMService(&cfg.GigaChat, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize LLM service", zap.Error(err))
	}
	defer llmService.Close()

	appLogger.Info("Starting database seeding...")

	// Seed knowledge base from PDF files
	seedDir := filepath.Join("cmd", "seed")
	cacheFile := filepath.Join(seedDir, ".seed_cache.json")
	if err := seedKnowledgeBaseFromPDFs(ctx, seedDir, cacheFile, knowledgeRepo, llmService, appLogger); err != nil {
		appLogger.Fatal("Failed to seed knowledge base from PDFs", zap.Error(err))
	}

	appLogger.Info("Database seeding completed successfully!")
}

// ProcessedFile represents a processed PDF file in cache
type ProcessedFile struct {
	FilePath    string    `json:"file_path"`
	FileHash    string    `json:"file_hash"`
	ProcessedAt time.Time `json:"processed_at"`
}

// CacheData stores information about processed files
type CacheData struct {
	ProcessedFiles map[string]ProcessedFile `json:"processed_files"` // key: file path
}

// loadCache loads the cache of processed files
func loadCache(cacheFile string) (*CacheData, error) {
	cache := &CacheData{
		ProcessedFiles: make(map[string]ProcessedFile),
	}

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return cache, nil
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	if len(data) == 0 {
		return cache, nil
	}

	if err := json.Unmarshal(data, cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	return cache, nil
}

// saveCache saves the cache of processed files
func saveCache(cacheFile string, cache *CacheData) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// calculateFileHash calculates MD5 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// seedKnowledgeBaseFromPDFs processes PDF files and creates knowledge base entries
func seedKnowledgeBaseFromPDFs(
	ctx context.Context,
	seedDir string,
	cacheFile string,
	repo *repository.KnowledgeRepository,
	llmService *service.LLMService,
	logger *zap.Logger,
) error {
	now := time.Now()

	// Load cache
	cache, err := loadCache(cacheFile)
	if err != nil {
		logger.Warn("Failed to load cache, will process all files", zap.Error(err))
		cache = &CacheData{ProcessedFiles: make(map[string]ProcessedFile)}
	}

	// Map PDF files to knowledge types
	pdfFiles := []struct {
		path     string
		kbType   models.KnowledgeType
		category string
		bank     string
	}{
		// Bank tariffs - Sberbank
		{"sberbank_tarifi.pdf", models.KnowledgeTypeBankTariff, "general", "Сбербанк"},
		{"sberbank_debetovye_karty_tarify.pdf", models.KnowledgeTypeBankTariff, "debit_cards", "Сбербанк"},
		{"sberbank_zarplatnye_karty_tarify.pdf", models.KnowledgeTypeBankTariff, "salary_cards", "Сбербанк"},
		{"sberbank_tarify_na_perevody.pdf", models.KnowledgeTypeBankTariff, "transfers", "Сбербанк"},
		{"sberbank_limity_na_operacii_mob.pdf", models.KnowledgeTypeBankTariff, "limits", "Сбербанк"},
		// Bank tariffs - Other banks
		{"alfabank_procent_na_ostatok_bs.pdf", models.KnowledgeTypeBankTariff, "savings", "Альфа-Банк"},
		{"tinkoff_bank_cashback.pdf", models.KnowledgeTypeBankTariff, "cashback", "Тинькофф"},
		// Government tariffs
		{"gos_poshlina.pdf", models.KnowledgeTypeGovTariff, "state_duty", ""},
		{"lgoti_fiz_litsa.pdf", models.KnowledgeTypeGovTariff, "benefits", ""},
		// Education
		{"uchebnik.pdf", models.KnowledgeTypeEducation, "textbook", ""},
		{"knif_fin_gramotnost.pdf", models.KnowledgeTypeEducation, "financial_literacy", ""},
	}

	for _, pdfInfo := range pdfFiles {
		pdfPath := filepath.Join(seedDir, pdfInfo.path)

		// Check if file exists
		if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
			logger.Warn("PDF file not found, skipping", zap.String("path", pdfPath))
			continue
		}

		// Calculate file hash
		fileHash, err := calculateFileHash(pdfPath)
		if err != nil {
			logger.Warn("Failed to calculate file hash, will process anyway", zap.String("path", pdfPath), zap.Error(err))
		}

		// Check if file was already processed
		if cached, exists := cache.ProcessedFiles[pdfPath]; exists {
			if cached.FileHash == fileHash {
				logger.Info("PDF file already processed, skipping",
					zap.String("path", pdfPath),
					zap.Time("processed_at", cached.ProcessedAt),
				)
				continue
			} else {
				logger.Info("PDF file changed, reprocessing",
					zap.String("path", pdfPath),
					zap.String("old_hash", cached.FileHash),
					zap.String("new_hash", fileHash),
				)
			}
		}

		logger.Info("Processing PDF file", zap.String("path", pdfPath))

		// Extract text from PDF using GigaChat Vision API
		text, err := extractTextFromPDF(ctx, pdfPath, llmService, logger)
		if err != nil {
			// Check if error is due to file size (413)
			if strings.Contains(err.Error(), "413") || strings.Contains(err.Error(), "too large") {
				logger.Warn("PDF file too large, skipping",
					zap.String("path", pdfPath),
					zap.Error(err),
				)
			} else {
				logger.Error("Failed to extract text from PDF", zap.String("path", pdfPath), zap.Error(err))
			}
			continue
		}

		if text == "" {
			logger.Warn("No text extracted from PDF", zap.String("path", pdfPath))
			continue
		}

		// Generate title from filename
		title := generateTitleFromFilename(pdfInfo.path, pdfInfo.bank, pdfInfo.category)

		// Create metadata
		metadata := map[string]interface{}{
			"source_file": pdfInfo.path,
			"category":    pdfInfo.category,
		}
		if pdfInfo.bank != "" {
			metadata["bank"] = pdfInfo.bank
		}

		metadataJSON, _ := json.Marshal(metadata)

		// Create knowledge base entry
		kb := &models.KnowledgeBase{
			ID:        uuid.New(),
			Type:      pdfInfo.kbType,
			Title:     title,
			Content:   text,
			Embedding: []float32{}, // Empty embedding for now
			Metadata:  string(metadataJSON),
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := repo.Create(ctx, kb); err != nil {
			logger.Error("Failed to create knowledge base entry", zap.String("path", pdfPath), zap.Error(err))
			continue
		}

		logger.Info("Created knowledge base entry from PDF",
			zap.String("title", title),
			zap.String("type", string(pdfInfo.kbType)),
			zap.Int("content_length", len(text)),
		)

		// Update cache
		cache.ProcessedFiles[pdfPath] = ProcessedFile{
			FilePath:    pdfPath,
			FileHash:    fileHash,
			ProcessedAt: now,
		}
	}

	// Save cache
	if err := saveCache(cacheFile, cache); err != nil {
		logger.Warn("Failed to save cache", zap.Error(err))
	} else {
		logger.Info("Cache saved", zap.Int("processed_files", len(cache.ProcessedFiles)))
	}

	return nil
}

// extractTextFromPDF extracts text from PDF using GigaChat Vision API
func extractTextFromPDF(
	ctx context.Context,
	pdfPath string,
	llmService *service.LLMService,
	logger *zap.Logger,
) (string, error) {
	// Open PDF file
	file, err := os.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Upload PDF to GigaChat
	fileName := filepath.Base(pdfPath)
	fileID, err := llmService.UploadFile(ctx, file, fileName)
	if err != nil {
		return "", fmt.Errorf("failed to upload PDF: %w", err)
	}

	logger.Info("PDF uploaded to GigaChat", zap.String("file_id", fileID), zap.String("file", fileName))

	// Use Vision API to extract text
	prompt := `Извлеки весь текст из этого документа. 
Верни только текст, который содержится в документе, без дополнительных комментариев.
Сохрани структуру документа (заголовки, списки, таблицы) в текстовом виде.
Если документ содержит таблицы, представь их в текстовом формате с описанием.`

	// Use direct HTTP API call for Vision
	text, err := llmService.ExtractTextViaVisionAPI(ctx, fileID, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to extract text via Vision API: %w", err)
	}

	return text, nil
}

// generateTitleFromFilename generates a human-readable title from filename
func generateTitleFromFilename(filename, bank, category string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Replace underscores with spaces
	name = strings.ReplaceAll(name, "_", " ")

	// Capitalize first letter of each word
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	title := strings.Join(words, " ")

	// Add bank name if available
	if bank != "" {
		title = bank + ": " + title
	}

	return title
}
