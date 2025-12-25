package models

import (
	"time"

	"github.com/google/uuid"
)

type DocumentType string

const (
	DocumentTypeReceipt   DocumentType = "receipt"
	DocumentTypeStatement DocumentType = "statement"
	DocumentTypeScreenshot DocumentType = "screenshot"
)

type Document struct {
	ID          uuid.UUID   `db:"id"`
	UserID      uuid.UUID   `db:"user_id"`
	Type        DocumentType `db:"type"`
	FileName    string      `db:"file_name"`
	FileSize    int64       `db:"file_size"`
	FileURL     string      `db:"file_url"`
	ExtractedText string    `db:"extracted_text"`
	CreatedAt   time.Time   `db:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at"`
}

