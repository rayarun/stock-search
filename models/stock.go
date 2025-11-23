package models

type Stock struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	Exchange        string  `json:"exchange"`
	Type            string  `json:"type"`
	Brand           string  `json:"brand"`
	PopularityScore float64 `json:"popularity_score"` // 0.0 to 1.0, used for ranking
}
