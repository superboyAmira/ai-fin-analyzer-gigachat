package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rag-iishka/internal/dto"
	"rag-iishka/internal/models"
	"rag-iishka/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DocumentService struct {
	docRepo    *repository.DocumentRepository
	txRepo     *repository.TransactionRepository
	recRepo    *repository.RecommendationRepository
	ocrService *OCRService
	llmService *LLMService
	recService *RecommendationService
	uploadDir  string
	logger     *zap.Logger
}

func NewDocumentService(
	docRepo *repository.DocumentRepository,
	txRepo *repository.TransactionRepository,
	recRepo *repository.RecommendationRepository,
	ocrService *OCRService,
	llmService *LLMService,
	recService *RecommendationService,
	uploadDir string,
	logger *zap.Logger,
) *DocumentService {
	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logger.Warn("Failed to create upload directory", zap.Error(err))
	}

	return &DocumentService{
		docRepo:    docRepo,
		txRepo:     txRepo,
		recRepo:    recRepo,
		ocrService: ocrService,
		llmService: llmService,
		recService: recService,
		uploadDir:  uploadDir,
		logger:     logger,
	}
}

// UploadDocument uploads and saves a document
func (s *DocumentService) UploadDocument(ctx context.Context, userID uuid.UUID, file io.Reader, fileName string, docType models.DocumentType) (*dto.DocumentResponse, error) {
	// Generate unique file name
	fileID := uuid.New()
	ext := filepath.Ext(fileName)
	newFileName := fileID.String() + ext
	filePath := filepath.Join(s.uploadDir, newFileName)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	fileSize, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create document record
	now := time.Now()
	doc := &models.Document{
		ID:        fileID,
		UserID:    userID,
		Type:      docType,
		FileName:  fileName,
		FileSize:  fileSize,
		FileURL:   "/uploads/" + newFileName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	return &dto.DocumentResponse{
		ID:        doc.ID.String(),
		Type:      string(doc.Type),
		FileName:  doc.FileName,
		FileSize:  doc.FileSize,
		FileURL:   doc.FileURL,
		CreatedAt: doc.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ProcessDocument processes a document: OCR -> LLM analysis -> RAG -> recommendations
func (s *DocumentService) ProcessDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) (*dto.ProcessDocumentResponse, error) {
	// 1. Get document
	doc, err := s.docRepo.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	if doc.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// 2. Extract text using OCR
	filePath := filepath.Join(s.uploadDir, filepath.Base(doc.FileURL))
	extractedText, err := s.ocrService.ExtractText(ctx, filePath)
	if err != nil {
		s.logger.Warn("OCR extraction failed", zap.Error(err))
		// Continue with empty text
		extractedText = ""
	}
	
	// Check if extracted text is an error message from LLM
	if extractedText != "" {
		textLower := strings.ToLower(extractedText)
		errorPhrases := []string{
			"не могу помочь",
			"не могу обработать",
			"предоставьте содержимое",
			"предоставь содержимое",
			"не могу извлечь",
			"cannot help",
			"cannot process",
			"please provide",
		}
		for _, phrase := range errorPhrases {
			if strings.Contains(textLower, phrase) {
				s.logger.Warn("OCR returned error message instead of text, treating as empty",
					zap.String("text", extractedText),
				)
				extractedText = ""
				break
			}
		}
	}

	// Update document with extracted text
	if extractedText != "" {
		if err := s.docRepo.UpdateExtractedText(ctx, documentID, extractedText); err != nil {
			s.logger.Warn("Failed to update extracted text", zap.Error(err))
		}
	}

	// 3. Analyze transactions using LLM
	var transactions []*models.Transaction
	if extractedText != "" {
		analyses, err := s.llmService.AnalyzeTransaction(ctx, extractedText)
		if err != nil {
			s.logger.Warn("LLM analysis failed", zap.Error(err))
		} else {
			// Convert analyses to transactions
			now := time.Now()
			for _, analysis := range analyses {
				tx := &models.Transaction{
					ID:             uuid.New(),
					DocumentID:     documentID,
					UserID:         userID,
					Amount:         analysis.Amount,
					Currency:       analysis.Currency,
					Description:    sanitizeUTF8(analysis.Description),
					Category:       analysis.Category,
					LLMDescription: sanitizeUTF8(analysis.LLMDescription),
					CreatedAt:      now,
					UpdatedAt:      now,
				}

				// Parse date if provided
				if analysis.Date != "" {
					if date, err := time.Parse("2006-01-02", analysis.Date); err == nil {
						tx.Date = date
					} else {
						tx.Date = now
					}
				} else {
					tx.Date = now
				}

				transactions = append(transactions, tx)
			}

			// Save transactions
			if len(transactions) > 0 {
				if err := s.txRepo.CreateBatch(ctx, transactions); err != nil {
					s.logger.Warn("Failed to save transactions", zap.Error(err))
				}
			}
		}
	}

	// 4. Generate recommendations for each transaction
	var allRecommendations []*models.Recommendation
	for _, tx := range transactions {
		recs, err := s.recService.GenerateRecommendations(ctx, tx, userID)
		if err != nil {
			s.logger.Warn("Failed to generate recommendations", zap.Error(err), zap.String("transaction_id", tx.ID.String()))
			continue
		}
		allRecommendations = append(allRecommendations, recs...)
	}

	// Save recommendations
	if len(allRecommendations) > 0 {
		if err := s.recRepo.CreateBatch(ctx, allRecommendations); err != nil {
			s.logger.Warn("Failed to save recommendations", zap.Error(err))
		}
	}

	// 5. Build response
	docResponse := &dto.DocumentResponse{
		ID:            doc.ID.String(),
		Type:          string(doc.Type),
		FileName:      doc.FileName,
		FileSize:      doc.FileSize,
		FileURL:       doc.FileURL,
		ExtractedText: extractedText,
		CreatedAt:     doc.CreatedAt.Format(time.RFC3339),
	}

	txResponses := make([]dto.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		txResponses[i] = dto.TransactionResponse{
			ID:             tx.ID.String(),
			Amount:         tx.Amount,
			Currency:       tx.Currency,
			Description:    tx.Description,
			Category:       string(tx.Category),
			LLMDescription: tx.LLMDescription,
			Date:           tx.Date.Format(time.RFC3339),
			CreatedAt:      tx.CreatedAt.Format(time.RFC3339),
		}
	}

	recResponses := make([]dto.RecommendationResponse, len(allRecommendations))
	for i, rec := range allRecommendations {
		recResponses[i] = dto.RecommendationResponse{
			ID:               rec.ID.String(),
			Title:            rec.Title,
			Description:      rec.Description,
			PotentialSavings: rec.PotentialSavings,
			Source:           rec.Source,
			CreatedAt:        rec.CreatedAt.Format(time.RFC3339),
		}
	}

	return &dto.ProcessDocumentResponse{
		Document:        *docResponse,
		Transactions:    txResponses,
		Recommendations: recResponses,
	}, nil
}

// ListDocuments lists user's documents
func (s *DocumentService) ListDocuments(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*dto.DocumentResponse, error) {
	docs, err := s.docRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.DocumentResponse, len(docs))
	for i, doc := range docs {
		responses[i] = &dto.DocumentResponse{
			ID:        doc.ID.String(),
			Type:      string(doc.Type),
			FileName:  doc.FileName,
			FileSize:  doc.FileSize,
			FileURL:   doc.FileURL,
			CreatedAt: doc.CreatedAt.Format(time.RFC3339),
		}
	}

	return responses, nil
}
