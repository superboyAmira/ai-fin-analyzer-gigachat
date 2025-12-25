package dto

type TransactionResponse struct {
	ID              string  `json:"id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	LLMDescription  string  `json:"llm_description"`
	Date            string  `json:"date"`
	CreatedAt       string  `json:"created_at"`
}

