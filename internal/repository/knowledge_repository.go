package repository

import (
	"context"
	"rag-iishka/internal/models"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
)

type KnowledgeRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewKnowledgeRepository(db *pgxpool.Pool, logger *zap.Logger) *KnowledgeRepository {
	return &KnowledgeRepository{
		db:     db,
		logger: logger,
	}
}

func (r *KnowledgeRepository) Create(ctx context.Context, kb *models.KnowledgeBase) error {
	// Convert []float32 to pgvector format
	embeddingArray := pgtype.FlatArray[float32]{}
	for _, v := range kb.Embedding {
		embeddingArray = append(embeddingArray, v)
	}

	query := squirrel.Insert("knowledge_base").
		Columns("id", "type", "title", "content", "embedding", "metadata", "created_at", "updated_at").
		Values(kb.ID, kb.Type, kb.Title, kb.Content, embeddingArray, kb.Metadata, kb.CreatedAt, kb.UpdatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *KnowledgeRepository) SearchSimilar(ctx context.Context, embedding []float32, topK int, knowledgeType *models.KnowledgeType) ([]*models.KnowledgeBase, error) {
	// Convert []float32 to pgvector format
	embeddingArray := pgtype.FlatArray[float32]{}
	for _, v := range embedding {
		embeddingArray = append(embeddingArray, v)
	}

	// Build query with cosine similarity
	query := squirrel.Select("id", "type", "title", "content", "embedding", "metadata", "created_at", "updated_at",
		"(embedding <=> $1) as distance").
		From("knowledge_base").
		OrderBy("distance ASC").
		Limit(uint64(topK)).
		PlaceholderFormat(squirrel.Dollar)

	if knowledgeType != nil {
		query = query.Where(squirrel.Eq{"type": *knowledgeType})
	}

	// Note: This is a simplified version. In production, you'd use pgvector's <=> operator
	// For now, we'll use a basic text search approach
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	// For now, return empty results as pgvector integration requires additional setup
	// In production, you would use: SELECT *, embedding <=> $1::vector AS distance FROM knowledge_base ORDER BY distance LIMIT $2
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.KnowledgeBase
	for rows.Next() {
		var kb models.KnowledgeBase
		var distance float64
		var embeddingData pgtype.FlatArray[float32]
		
		if err := rows.Scan(
			&kb.ID, &kb.Type, &kb.Title, &kb.Content, &embeddingData, &kb.Metadata, &kb.CreatedAt, &kb.UpdatedAt, &distance,
		); err != nil {
			return nil, err
		}
		
		kb.Embedding = []float32(embeddingData)
		results = append(results, &kb)
	}

	return results, nil
}

// SimpleTextSearch performs a basic text search (fallback when embeddings are not available)
func (r *KnowledgeRepository) SimpleTextSearch(ctx context.Context, queryText string, topK int, knowledgeType *models.KnowledgeType) ([]*models.KnowledgeBase, error) {
	query := squirrel.Select("id", "type", "title", "content", "embedding", "metadata", "created_at", "updated_at").
		From("knowledge_base").
		Where(squirrel.Or{
			squirrel.ILike{"title": "%" + queryText + "%"},
			squirrel.ILike{"content": "%" + queryText + "%"},
		}).
		Limit(uint64(topK)).
		PlaceholderFormat(squirrel.Dollar)

	if knowledgeType != nil {
		query = query.Where(squirrel.Eq{"type": *knowledgeType})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.KnowledgeBase
	for rows.Next() {
		var kb models.KnowledgeBase
		var embeddingData pgtype.FlatArray[float32]
		
		if err := rows.Scan(
			&kb.ID, &kb.Type, &kb.Title, &kb.Content, &embeddingData, &kb.Metadata, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		
		kb.Embedding = []float32(embeddingData)
		results = append(results, &kb)
	}

	return results, nil
}

// Helper function to calculate cosine similarity (for fallback)
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

