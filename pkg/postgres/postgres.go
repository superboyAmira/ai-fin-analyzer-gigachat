package postgres

import (
	"context"
	"fmt"

	"rag-iishka/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPool(ctx context.Context, cfg *config.DatabaseConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.String("database", cfg.DBName),
	)

	return pool, nil
}

