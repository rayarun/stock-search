package models

type Stock struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	Exchange        string  `json:"exchange"`
	Type            string  `json:"type"`
	Brand           string  `json:"brand"`
	Sector          string  `json:"sector"`           // e.g., "Banking", "IT", "Pharma", "Broking"
	Industry        string  `json:"industry"`         // e.g., "Software Services", "Private Banks"
	Tags            string  `json:"tags"`             // Searchable keywords, comma-separated
	PopularityScore float64 `json:"popularity_score"` // 0.0 to 1.0, used for ranking
}
