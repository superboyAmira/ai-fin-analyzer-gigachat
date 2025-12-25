package dto

type RecommendationResponse struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	PotentialSavings float64 `json:"potential_savings"`
	Source          string  `json:"source"`
	CreatedAt       string  `json:"created_at"`
}

