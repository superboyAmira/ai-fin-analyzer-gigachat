package dto

type UploadDocumentRequest struct {
	Type string `json:"type" validate:"required,oneof=receipt statement screenshot"`
}

type DocumentResponse struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	FileURL       string `json:"file_url"`
	ExtractedText string `json:"extracted_text,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type ProcessDocumentResponse struct {
	Document      DocumentResponse   `json:"document"`
	Transactions  []TransactionResponse `json:"transactions"`
	Recommendations []RecommendationResponse `json:"recommendations"`
}

