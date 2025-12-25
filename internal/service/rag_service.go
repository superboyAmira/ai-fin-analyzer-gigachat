package service

import (
	"context"
	"fmt"
	"strings"

	"rag-iishka/internal/models"
	"rag-iishka/internal/repository"
	"rag-iishka/pkg/config"

	"go.uber.org/zap"
)

type RAGService struct {
	knowledgeRepo *repository.KnowledgeRepository
	config        *config.RAGConfig
	logger        *zap.Logger
}

func NewRAGService(knowledgeRepo *repository.KnowledgeRepository, cfg *config.RAGConfig, logger *zap.Logger) *RAGService {
	return &RAGService{
		knowledgeRepo: knowledgeRepo,
		config:        cfg,
		logger:        logger,
	}
}

// SearchKnowledge searches for relevant knowledge base entries
// In production, you would generate embeddings for the query and use vector search
func (s *RAGService) SearchKnowledge(ctx context.Context, query string, transactionCategory *models.TransactionCategory) ([]*models.KnowledgeBase, error) {
	// Map transaction category to knowledge type
	var knowledgeType *models.KnowledgeType
	if transactionCategory != nil {
		// For now, search all types. In production, you might want to map categories to specific knowledge types
		// For example: food -> bank_tariff, utilities -> gov_tariff, etc.
	}

	// Try vector search first (if embeddings are available)
	// For now, fallback to text search
	results, err := s.knowledgeRepo.SimpleTextSearch(ctx, query, s.config.TopK, knowledgeType)
	if err != nil {
		s.logger.Warn("Vector search failed, using text search", zap.Error(err))
		// Fallback to text search
		results, err = s.knowledgeRepo.SimpleTextSearch(ctx, query, s.config.TopK, knowledgeType)
		if err != nil {
			return nil, fmt.Errorf("failed to search knowledge base: %w", err)
		}
	}

	s.logger.Info("Knowledge search completed", 
		zap.String("query", query),
		zap.Int("results", len(results)),
	)

	return results, nil
}

// BuildContext builds a context string from knowledge base results
func (s *RAGService) BuildContext(results []*models.KnowledgeBase) string {
	if len(results) == 0 {
		return "Нет релевантной информации в базе знаний."
	}

	var builder strings.Builder
	builder.WriteString("Релевантная информация из базы знаний:\n\n")

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, result.Type, result.Title))
		builder.WriteString(fmt.Sprintf("   %s\n\n", result.Content))
	}

	return builder.String()
}

// GenerateQueryFromTransaction generates a search query from transaction data
func (s *RAGService) GenerateQueryFromTransaction(transaction *models.Transaction) string {
	query := transaction.Description
	if transaction.LLMDescription != "" {
		query += " " + transaction.LLMDescription
	}
	query += " " + string(transaction.Category)
	return query
}

