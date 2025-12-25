package models

import (
	"time"

	"github.com/google/uuid"
)

type TransactionCategory string

const (
	CategoryFood        TransactionCategory = "food"
	CategoryTransport   TransactionCategory = "transport"
	CategoryUtilities   TransactionCategory = "utilities"
	CategoryShopping    TransactionCategory = "shopping"
	CategoryEntertainment TransactionCategory = "entertainment"
	CategoryHealthcare  TransactionCategory = "healthcare"
	CategoryEducation   TransactionCategory = "education"
	CategoryOther       TransactionCategory = "other"
)

type Transaction struct {
	ID          uuid.UUID          `db:"id"`
	DocumentID  uuid.UUID          `db:"document_id"`
	UserID      uuid.UUID          `db:"user_id"`
	Amount      float64            `db:"amount"`
	Currency    string             `db:"currency"`
	Description string             `db:"description"`
	Category    TransactionCategory `db:"category"`
	LLMDescription string          `db:"llm_description"`
	Date        time.Time          `db:"date"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
}

