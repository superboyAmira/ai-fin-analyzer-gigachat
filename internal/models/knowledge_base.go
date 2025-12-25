package models

import (
	"time"

	"github.com/google/uuid"
)

type KnowledgeType string

const (
	KnowledgeTypeBankTariff KnowledgeType = "bank_tariff"
	KnowledgeTypeGovTariff  KnowledgeType = "gov_tariff"
	KnowledgeTypeEducation  KnowledgeType = "education"
)

type KnowledgeBase struct {
	ID          uuid.UUID    `db:"id"`
	Type        KnowledgeType `db:"type"`
	Title       string       `db:"title"`
	Content     string       `db:"content"`
	Embedding   []float32    `db:"embedding"` // векторное представление
	Metadata    string       `db:"metadata"`  // JSON с дополнительными данными
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
}

