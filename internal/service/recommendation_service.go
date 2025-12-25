package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"rag-iishka/internal/models"
	"rag-iishka/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RecommendationService struct {
	llmService *LLMService
	ragService *RAGService
	recRepo    *repository.RecommendationRepository
	logger     *zap.Logger
}

func NewRecommendationService(
	llmService *LLMService,
	ragService *RAGService,
	recRepo *repository.RecommendationRepository,
	logger *zap.Logger,
) *RecommendationService {
	return &RecommendationService{
		llmService: llmService,
		ragService: ragService,
		recRepo:    recRepo,
		logger:     logger,
	}
}

// GenerateRecommendations generates recommendations for a transaction
func (s *RecommendationService) GenerateRecommendations(
	ctx context.Context,
	transaction *models.Transaction,
	userID uuid.UUID,
) ([]*models.Recommendation, error) {
	// 1. Search knowledge base
	query := s.ragService.GenerateQueryFromTransaction(transaction)
	knowledgeResults, err := s.ragService.SearchKnowledge(ctx, query, &transaction.Category)
	if err != nil {
		s.logger.Warn("Failed to search knowledge base", zap.Error(err))
		knowledgeResults = []*models.KnowledgeBase{}
	}

	// 2. Build context from knowledge base
	context := s.ragService.BuildContext(knowledgeResults)

	// 3. Generate recommendation using LLM
	// Convert transaction to analysis format for LLM
	transactionAnalysis := &TransactionAnalysis{
		Description:    transaction.Description,
		Category:       transaction.Category,
		Amount:         transaction.Amount,
		Currency:       transaction.Currency,
		LLMDescription: transaction.LLMDescription,
		Date:           transaction.Date.Format("2006-01-02"),
	}

	llmResponse, err := s.llmService.GenerateRecommendationPrompt(ctx, transactionAnalysis, context)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendation: %w", err)
	}

	// 4. Parse recommendations from LLM response
	recommendations := s.parseRecommendations(llmResponse, transaction.ID, userID, knowledgeResults)

	s.logger.Info("Recommendations generated",
		zap.String("transaction_id", transaction.ID.String()),
		zap.Int("count", len(recommendations)),
	)

	return recommendations, nil
}

// parseRecommendations parses recommendations from LLM response
func (s *RecommendationService) parseRecommendations(
	llmResponse string,
	transactionID uuid.UUID,
	userID uuid.UUID,
	knowledgeResults []*models.KnowledgeBase,
) []*models.Recommendation {
	var recommendations []*models.Recommendation

	// Simple parsing: split by numbered list or bullet points
	lines := strings.Split(llmResponse, "\n")
	currentRec := &models.Recommendation{
		ID:            uuid.New(),
		TransactionID: transactionID,
		UserID:        userID,
		Source:        "llm",
	}

	var currentText strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line starts a new recommendation (numbered or bulleted)
		if matched, _ := regexp.MatchString(`^(\d+[\.\)]|[-*])\s+`, line); matched {
			// Save previous recommendation if exists
			if currentText.Len() > 0 {
				currentRec.Description = sanitizeUTF8(currentText.String())
				currentRec.Title = sanitizeUTF8(s.extractTitle(currentRec.Description))
				currentRec.PotentialSavings = s.extractSavings(currentRec.Description, knowledgeResults)
				recommendations = append(recommendations, currentRec)
			}

			// Start new recommendation
			currentRec = &models.Recommendation{
				ID:            uuid.New(),
				TransactionID: transactionID,
				UserID:        userID,
				Source:        s.getSourceFromKnowledge(knowledgeResults),
			}
			currentText.Reset()
			line = regexp.MustCompile(`^(\d+[\.\)]|[-*])\s+`).ReplaceAllString(line, "")
		}

		currentText.WriteString(line)
		currentText.WriteString(" ")
	}

	// Save last recommendation
	if currentText.Len() > 0 {
		currentRec.Description = sanitizeUTF8(currentText.String())
		currentRec.Title = sanitizeUTF8(s.extractTitle(currentRec.Description))
		currentRec.PotentialSavings = s.extractSavings(currentRec.Description, knowledgeResults)
		recommendations = append(recommendations, currentRec)
	}

	// If no structured recommendations found, create one from entire response
	if len(recommendations) == 0 {
		recommendations = append(recommendations, &models.Recommendation{
			ID:               uuid.New(),
			TransactionID:    transactionID,
			UserID:           userID,
			Title:            sanitizeUTF8("Рекомендация по сокращению расходов"),
			Description:      sanitizeUTF8(llmResponse),
			PotentialSavings: s.extractSavings(llmResponse, knowledgeResults),
			Source:           s.getSourceFromKnowledge(knowledgeResults),
		})
	}

	return recommendations
}

func (s *RecommendationService) extractTitle(description string) string {
	// Extract first sentence or first 50 characters as title
	sentences := strings.Split(description, ".")
	if len(sentences) > 0 {
		title := strings.TrimSpace(sentences[0])
		if len(title) > 100 {
			title = title[:100] + "..."
		}
		return title
	}

	if len(description) > 100 {
		return description[:100] + "..."
	}
	return description
}

func (s *RecommendationService) extractSavings(description string, knowledgeResults []*models.KnowledgeBase) float64 {
	// Try to extract potential savings amount from description
	// Look for patterns like "сэкономить 1000 руб", "экономия 5%", etc.

	// Pattern 1: "X руб" or "X рублей"
	re1 := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:руб|рублей|RUB)`)
	matches := re1.FindStringSubmatch(description)
	if len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return amount
		}
	}

	// Pattern 2: "X%" - calculate percentage of transaction amount
	// This would require transaction amount, so we'll skip for now

	return 0.0
}

func (s *RecommendationService) getSourceFromKnowledge(knowledgeResults []*models.KnowledgeBase) string {
	if len(knowledgeResults) > 0 {
		return string(knowledgeResults[0].Type)
	}
	return "llm"
}
