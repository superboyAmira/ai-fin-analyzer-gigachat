package repository

import (
	"context"
	"rag-iishka/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
)

type DocumentRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewDocumentRepository(db *pgxpool.Pool, logger *zap.Logger) *DocumentRepository {
	return &DocumentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *DocumentRepository) Create(ctx context.Context, doc *models.Document) error {
	query := squirrel.Insert("documents").
		Columns("id", "user_id", "type", "file_name", "file_size", "file_url", "extracted_text", "created_at", "updated_at").
		Values(doc.ID, doc.UserID, doc.Type, doc.FileName, doc.FileSize, doc.FileURL, doc.ExtractedText, doc.CreatedAt, doc.UpdatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *DocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	query := squirrel.Select("id", "user_id", "type", "file_name", "file_size", "file_url", "extracted_text", "created_at", "updated_at").
		From("documents").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var doc models.Document
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&doc.ID, &doc.UserID, &doc.Type, &doc.FileName, &doc.FileSize, &doc.FileURL, &doc.ExtractedText, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *DocumentRepository) UpdateExtractedText(ctx context.Context, id uuid.UUID, text string) error {
	query := squirrel.Update("documents").
		Set("extracted_text", text).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *DocumentRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Document, error) {
	query := squirrel.Select("id", "user_id", "type", "file_name", "file_size", "file_url", "extracted_text", "created_at", "updated_at").
		From("documents").
		Where(squirrel.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
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

	var documents []*models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(
			&doc.ID, &doc.UserID, &doc.Type, &doc.FileName, &doc.FileSize, &doc.FileURL, &doc.ExtractedText, &doc.CreatedAt, &doc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		documents = append(documents, &doc)
	}

	return documents, nil
}

