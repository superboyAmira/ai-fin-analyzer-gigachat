package models

import (
	"time"

	"github.com/google/uuid"
)

type Recommendation struct {
	ID            uuid.UUID `db:"id"`
	TransactionID uuid.UUID `db:"transaction_id"`
	UserID        uuid.UUID `db:"user_id"`
	Title         string    `db:"title"`
	Description   string    `db:"description"`
	PotentialSavings float64 `db:"potential_savings"`
	Source        string    `db:"source"` // источник из базы знаний
	CreatedAt     time.Time `db:"created_at"`
}

