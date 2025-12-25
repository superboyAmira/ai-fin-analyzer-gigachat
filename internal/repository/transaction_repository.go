package repository

import (
	"context"
	"rag-iishka/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
)

type TransactionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewTransactionRepository(db *pgxpool.Pool, logger *zap.Logger) *TransactionRepository {
	return &TransactionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := squirrel.Insert("transactions").
		Columns("id", "document_id", "user_id", "amount", "currency", "description", "category", "llm_description", "date", "created_at", "updated_at").
		Values(tx.ID, tx.DocumentID, tx.UserID, tx.Amount, tx.Currency, tx.Description, tx.Category, tx.LLMDescription, tx.Date, tx.CreatedAt, tx.UpdatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *TransactionRepository) CreateBatch(ctx context.Context, transactions []*models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	builder := squirrel.Insert("transactions").
		Columns("id", "document_id", "user_id", "amount", "currency", "description", "category", "llm_description", "date", "created_at", "updated_at").
		PlaceholderFormat(squirrel.Dollar)

	for _, tx := range transactions {
		builder = builder.Values(tx.ID, tx.DocumentID, tx.UserID, tx.Amount, tx.Currency, tx.Description, tx.Category, tx.LLMDescription, tx.Date, tx.CreatedAt, tx.UpdatedAt)
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *TransactionRepository) GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]*models.Transaction, error) {
	query := squirrel.Select("id", "document_id", "user_id", "amount", "currency", "description", "category", "llm_description", "date", "created_at", "updated_at").
		From("transactions").
		Where(squirrel.Eq{"document_id": documentID}).
		OrderBy("date DESC").
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

	var transactions []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID, &tx.DocumentID, &tx.UserID, &tx.Amount, &tx.Currency, &tx.Description, &tx.Category, &tx.LLMDescription, &tx.Date, &tx.CreatedAt, &tx.UpdatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, &tx)
	}

	return transactions, nil
}

