package search

import (
	"stock-search/models"
	"strings"
)

type SearchEngine interface {
	Search(query string) []models.Stock
	GetBySymbol(symbol string) *models.Stock
	GetStock(symbol, exchange string) *models.Stock
}

type InMemoryEngine struct {
	stocks []models.Stock
}

func NewInMemoryEngine(stocks []models.Stock) *InMemoryEngine {
	return &InMemoryEngine{stocks: stocks}
}

func (e *InMemoryEngine) Search(query string) []models.Stock {
	var results []models.Stock
	q := strings.ToLower(query)
	for _, stock := range e.stocks {
		if strings.HasPrefix(strings.ToLower(stock.Symbol), q) ||
			strings.Contains(strings.ToLower(stock.Name), q) {
			results = append(results, stock)
		}
	}
	return results
}

func (e *InMemoryEngine) GetBySymbol(symbol string) *models.Stock {
	for _, stock := range e.stocks {
		if strings.EqualFold(stock.Symbol, symbol) {
			return &stock
		}
	}
	return nil
}

func (e *InMemoryEngine) GetStock(symbol, exchange string) *models.Stock {
	for _, stock := range e.stocks {
		if strings.EqualFold(stock.Symbol, symbol) && strings.EqualFold(stock.Exchange, exchange) {
			return &stock
		}
	}
	// Fallback to GetBySymbol if exchange doesn't match
	return e.GetBySymbol(symbol)
}
