package loader

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"stock-search/models"
	"strings"
)

// CalculatePopularityScore assigns a popularity score based on well-known stocks
// Score ranges from 0.1 (unknown) to 1.0 (highly popular)
func CalculatePopularityScore(symbol string) float64 {
	symbol = strings.ToUpper(symbol)

	// Tier 1: Most popular stocks (0.9-1.0)
	tier1 := map[string]float64{
		"RELIANCE":   1.0,
		"TCS":        0.98,
		"HDFCBANK":   0.96,
		"INFY":       0.95,
		"ICICIBANK":  0.94,
		"HINDUNILVR": 0.93,
		"ITC":        0.92,
		"SBIN":       0.91,
		"BHARTIARTL": 0.90,
		"KOTAKBANK":  0.90,
	}

	// Tier 2: Well-known large caps (0.7-0.89)
	tier2 := map[string]float64{
		"BAJFINANCE": 0.85,
		"LT":         0.84,
		"ASIANPAINT": 0.83,
		"AXISBANK":   0.82,
		"MARUTI":     0.81,
		"SUNPHARMA":  0.80,
		"TITAN":      0.79,
		"NESTLEIND":  0.78,
		"ULTRACEMCO": 0.77,
		"WIPRO":      0.76,
		"TATAMOTORS": 0.75,
		"TATAPOWER":  0.74,
		"TATASTEEL":  0.73,
		"ADANIPORTS": 0.72,
		"ADANIENT":   0.71,
		"ONGC":       0.70,
	}

	// Tier 3: Mid-caps and sector leaders (0.4-0.69)
	tier3 := map[string]float64{
		"DIVISLAB":   0.65,
		"DRREDDY":    0.64,
		"CIPLA":      0.63,
		"TECHM":      0.62,
		"HCLTECH":    0.61,
		"POWERGRID":  0.60,
		"NTPC":       0.59,
		"COALINDIA":  0.58,
		"BPCL":       0.57,
		"IOC":        0.56,
		"GRASIM":     0.55,
		"JSWSTEEL":   0.54,
		"HINDALCO":   0.53,
		"VEDL":       0.52,
		"INDUSINDBK": 0.51,
		"BAJAJFINSV": 0.50,
		"M&M":        0.49,
		"EICHERMOT":  0.48,
		"HEROMOTOCO": 0.47,
		"BRITANNIA":  0.46,
		"SHREECEM":   0.45,
		"UPL":        0.44,
		"APOLLOHOSP": 0.43,
		"PIDILITIND": 0.42,
		"GODREJCP":   0.41,
		"DABUR":      0.40,
	}

	if score, ok := tier1[symbol]; ok {
		return score
	}
	if score, ok := tier2[symbol]; ok {
		return score
	}
	if score, ok := tier3[symbol]; ok {
		return score
	}

	// Default score for other stocks
	return 0.2
}

func LoadStocks(filePath string) ([]models.Stock, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var stocks []models.Stock
	// Skip header row if present (assuming first row is header)
	if len(records) > 0 {
		// Simple check: if the first cell is "Symbol", skip it
		if records[0][0] == "Symbol" {
			records = records[1:]
		}
	}

	for _, record := range records {
		if len(record) < 4 {
			continue
		}
		stock := models.Stock{
			Symbol:          record[0],
			Name:            record[1],
			Exchange:        record[2],
			Type:            record[3],
			Brand:           record[4],
			PopularityScore: CalculatePopularityScore(record[0]),
		}
		stocks = append(stocks, stock)
	}

	return stocks, nil
}

func LoadNSEStocks(filePath string) ([]models.Stock, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var stocks []models.Stock
	// Skip header row
	for i, record := range records {
		if i == 0 {
			continue
		}
		// NSE CSV Format: SYMBOL,NAME OF COMPANY, SERIES, DATE OF LISTING, PAID UP VALUE, MARKET LOT, ISIN NUMBER, FACE VALUE
		// We only want EQ series for now, or maybe all? Let's filter for EQ/BE to be safe, or just take all.
		// Let's take all for now.

		stock := models.Stock{
			Symbol:          record[0],
			Name:            record[1],
			Exchange:        "NSE",
			Type:            "Stock", // Defaulting to Stock
			Brand:           "",      // No brand data in this file
			PopularityScore: CalculatePopularityScore(record[0]),
		}
		stocks = append(stocks, stock)
	}

	// Apply automatic classification
	classifier := NewStockClassifier()
	stocks = classifier.ClassifyBatch(stocks)

	return stocks, nil
}

func LoadBSEStocks(filePath string) ([]models.Stock, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var stocks []models.Stock
	// Skip header row
	for i, record := range records {
		if i == 0 {
			continue
		}
		// BSE CSV Format: SYMBOL (Scrip Code),NAME OF COMPANY
		// We only want EQ series for now, or maybe all? Let's filter for EQ/BE to be safe, or just take all.
		// Let's take all for now.

		stock := models.Stock{
			Symbol:          record[0],
			Name:            record[1],
			Exchange:        "BSE",
			Type:            "Stock", // Defaulting to Stock
			Brand:           "",      // No brand data in this file
			PopularityScore: CalculatePopularityScore(record[0]),
		}
		stocks = append(stocks, stock)
	}

	// Apply automatic classification
	classifier := NewStockClassifier()
	stocks = classifier.ClassifyBatch(stocks)

	return stocks, nil
}

func LoadBrandMappings(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mappings map[string]string
	if err := json.NewDecoder(file).Decode(&mappings); err != nil {
		return nil, err
	}

	return mappings, nil
}
