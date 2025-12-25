package repository

import (
	"context"
	"rag-iishka/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
)

type UserRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewUserRepository(db *pgxpool.Pool, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := squirrel.Insert("users").
		Columns("id", "username", "email", "password", "created_at", "updated_at").
		Values(user.ID, user.Username, user.Email, user.Password, user.CreatedAt, user.UpdatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := squirrel.Select("id", "username", "email", "password", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"email": email}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := squirrel.Select("id", "username", "email", "password", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

