package repository

import (
	"context"
	"rag-iishka/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
)

type RecommendationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewRecommendationRepository(db *pgxpool.Pool, logger *zap.Logger) *RecommendationRepository {
	return &RecommendationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *RecommendationRepository) Create(ctx context.Context, rec *models.Recommendation) error {
	query := squirrel.Insert("recommendations").
		Columns("id", "transaction_id", "user_id", "title", "description", "potential_savings", "source", "created_at").
		Values(rec.ID, rec.TransactionID, rec.UserID, rec.Title, rec.Description, rec.PotentialSavings, rec.Source, rec.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *RecommendationRepository) CreateBatch(ctx context.Context, recommendations []*models.Recommendation) error {
	if len(recommendations) == 0 {
		return nil
	}

	builder := squirrel.Insert("recommendations").
		Columns("id", "transaction_id", "user_id", "title", "description", "potential_savings", "source", "created_at").
		PlaceholderFormat(squirrel.Dollar)

	for _, rec := range recommendations {
		builder = builder.Values(rec.ID, rec.TransactionID, rec.UserID, rec.Title, rec.Description, rec.PotentialSavings, rec.Source, rec.CreatedAt)
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *RecommendationRepository) GetByTransactionID(ctx context.Context, transactionID uuid.UUID) ([]*models.Recommendation, error) {
	query := squirrel.Select("id", "transaction_id", "user_id", "title", "description", "potential_savings", "source", "created_at").
		From("recommendations").
		Where(squirrel.Eq{"transaction_id": transactionID}).
		OrderBy("potential_savings DESC").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recommendations []*models.Recommendation
	for rows.Next() {
		var rec models.Recommendation
		if err := rows.Scan(
			&rec.ID, &rec.TransactionID, &rec.UserID, &rec.Title, &rec.Description, &rec.PotentialSavings, &rec.Source, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		recommendations = append(recommendations, &rec)
	}

	return recommendations, nil
}

