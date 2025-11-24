package search

import (
	"fmt"
	"log"
	"stock-search/models"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

type BleveEngine struct {
	index    bleve.Index
	semantic *SemanticSearch
}

func NewBleveEngine(indexPath string, stocks []models.Stock, semanticMappingsPath string) (*BleveEngine, error) {
	// Initialize semantic search
	semantic, err := NewSemanticSearch(semanticMappingsPath)
	if err != nil {
		log.Printf("Warning: Failed to load semantic search: %v", err)
		// Continue without semantic search
	}

	// Enrich stocks with sector/industry data from semantic mappings
	if semantic != nil {
		for i := range stocks {
			if stocks[i].Sector == "" {
				stocks[i].Sector = semantic.GetSectorForSymbol(stocks[i].Symbol)
			}
			if stocks[i].Industry == "" {
				stocks[i].Industry = semantic.GetIndustryForSymbol(stocks[i].Symbol)
			}
		}
	}
	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index if not exists
		indexMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %v", err)
		}

		// Index data
		log.Println("Indexing stocks...")
		batch := index.NewBatch()
		for _, stock := range stocks {
			// Use Symbol as ID, but we might have duplicates across exchanges (e.g. RELIANCE on NSE and BSE)
			// So we need a unique ID.
			id := fmt.Sprintf("%s-%s", stock.Symbol, stock.Exchange)
			if err := batch.Index(id, stock); err != nil {
				return nil, fmt.Errorf("failed to add to batch: %v", err)
			}
		}
		if err := index.Batch(batch); err != nil {
			return nil, fmt.Errorf("failed to execute batch: %v", err)
		}
		log.Println("Indexing complete.")
	} else if err != nil {
		return nil, fmt.Errorf("failed to open index: %v", err)
	} else {
		log.Println("Opened existing index.")
	}

	return &BleveEngine{
		index:    index,
		semantic: semantic,
	}, nil
}

func buildIndexMapping() mapping.IndexMapping {
	// Create index mapping with custom field configurations
	indexMapping := bleve.NewIndexMapping()

	// Create a stock mapping
	stockMapping := bleve.NewDocumentMapping()

	// Add numeric field mapping for PopularityScore
	// This allows us to use it in scoring/sorting
	popularityFieldMapping := bleve.NewNumericFieldMapping()
	popularityFieldMapping.Store = true
	popularityFieldMapping.Index = true
	stockMapping.AddFieldMappingsAt("popularity_score", popularityFieldMapping)
	stockMapping.AddFieldMappingsAt("popularityscore", popularityFieldMapping)

	// Add text field mappings for semantic search fields
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = true
	textFieldMapping.Index = true
	stockMapping.AddFieldMappingsAt("sector", textFieldMapping)
	stockMapping.AddFieldMappingsAt("industry", textFieldMapping)
	stockMapping.AddFieldMappingsAt("tags", textFieldMapping)

	indexMapping.AddDocumentMapping("_default", stockMapping)

	return indexMapping
}

func (e *BleveEngine) Search(query string) []models.Stock {
	// Check if this is a semantic query
	if e.semantic != nil && e.semantic.IsSemanticQuery(query) {
		return e.semanticSearch(query)
	}

	// Use regular search for non-semantic queries
	return e.regularSearch(query)
}

// semanticSearch handles natural language queries like "top broking stocks"
func (e *BleveEngine) semanticSearch(query string) []models.Stock {
	// Extract sectors from the query
	sectors := e.semantic.ExtractSectors(query)

	if len(sectors) == 0 {
		// No sectors found, fall back to regular search
		return e.regularSearch(query)
	}

	// Get stock symbols for the matched sectors
	targetSymbols := e.semantic.GetStockSymbolsForSectors(sectors)

	if len(targetSymbols) == 0 {
		return []models.Stock{}
	}

	// Build queries for each symbol and combine them
	if len(targetSymbols) == 0 {
		return []models.Stock{}
	}

	// Create first query
	firstQuery := bleve.NewTermQuery(strings.ToLower(targetSymbols[0]))
	firstQuery.SetField("symbol")
	searchQuery := bleve.NewDisjunctionQuery(firstQuery)

	// Add remaining queries
	for i := 1; i < len(targetSymbols); i++ {
		termQuery := bleve.NewTermQuery(strings.ToLower(targetSymbols[i]))
		termQuery.SetField("symbol")
		searchQuery.AddQuery(termQuery)
	}

	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Fields = []string{"symbol", "name", "exchange", "type", "brand", "sector", "industry", "tags", "popularity_score"}
	searchRequest.Size = 100

	searchResults, err := e.index.Search(searchRequest)
	if err != nil {
		log.Printf("Semantic search error: %v", err)
		return []models.Stock{}
	}

	// Helper functions
	getString := func(fields map[string]interface{}, key string) string {
		if val, ok := fields[key].(string); ok {
			return val
		}
		return ""
	}

	getFloat := func(fields map[string]interface{}, key string) float64 {
		if val, ok := fields[key].(float64); ok {
			return val
		}
		return 0.0
	}

	// Build results with scores
	type ScoredStock struct {
		Stock      models.Stock
		FinalScore float64
	}

	var scoredResults []ScoredStock
	for _, hit := range searchResults.Hits {
		popularityScore := getFloat(hit.Fields, "popularity_score")

		stock := models.Stock{
			Symbol:          getString(hit.Fields, "symbol"),
			Name:            getString(hit.Fields, "name"),
			Exchange:        getString(hit.Fields, "exchange"),
			Type:            getString(hit.Fields, "type"),
			Brand:           getString(hit.Fields, "brand"),
			Sector:          getString(hit.Fields, "sector"),
			Industry:        getString(hit.Fields, "industry"),
			Tags:            getString(hit.Fields, "tags"),
			PopularityScore: popularityScore,
		}

		// For semantic search, prioritize popularity score
		// since all results are equally relevant (from same sector)
		scoredResults = append(scoredResults, ScoredStock{
			Stock:      stock,
			FinalScore: popularityScore,
		})
	}

	// Sort by popularity score (descending)
	for i := 0; i < len(scoredResults); i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].FinalScore > scoredResults[i].FinalScore {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	// Extract just the stocks
	var results []models.Stock
	for _, scored := range scoredResults {
		results = append(results, scored.Stock)
	}

	return results
}

// regularSearch is the original search logic extracted for reuse
func (e *BleveEngine) regularSearch(query string) []models.Stock {
	// Advanced search with match-type boosting and popularity ranking

	// 1. Exact Symbol Match (highest priority, boost = 10.0)
	exactQuery := bleve.NewTermQuery(strings.ToLower(query))
	exactQuery.SetField("symbol")
	exactQuery.SetBoost(10.0)

	// 2. Prefix Symbol Match (high priority, boost = 5.0)
	prefixQuery := bleve.NewPrefixQuery(strings.ToLower(query))
	prefixQuery.SetField("symbol")
	prefixQuery.SetBoost(5.0)

	// 3. Match Query on Name (medium priority, boost = 3.0)
	nameMatchQuery := bleve.NewMatchQuery(query)
	nameMatchQuery.SetField("name")
	nameMatchQuery.SetBoost(3.0)

	// 4. Wildcard Query for Symbol (fuzzy search, boost = 2.0)
	wildcardSymbol := bleve.NewWildcardQuery("*" + strings.ToLower(query) + "*")
	wildcardSymbol.SetField("symbol")
	wildcardSymbol.SetBoost(2.0)

	// 5. Wildcard Query for Name (fuzzy search, boost = 1.5)
	wildcardName := bleve.NewWildcardQuery("*" + strings.ToLower(query) + "*")
	wildcardName.SetField("name")
	wildcardName.SetBoost(1.5)

	// 6. Wildcard Query for Brand (fuzzy search, boost = 1.0)
	wildcardBrand := bleve.NewWildcardQuery("*" + strings.ToLower(query) + "*")
	wildcardBrand.SetField("brand")
	wildcardBrand.SetBoost(1.0)

	// Combine all queries with Disjunction (OR)
	searchQuery := bleve.NewDisjunctionQuery(
		exactQuery,
		prefixQuery,
		nameMatchQuery,
		wildcardSymbol,
		wildcardName,
		wildcardBrand,
	)

	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Fields = []string{"symbol", "name", "exchange", "type", "brand", "sector", "industry", "tags", "popularity_score"}
	searchRequest.Size = 100 // Get more results for better ranking

	searchResults, err := e.index.Search(searchRequest)
	if err != nil {
		log.Printf("Search error: %v", err)
		return []models.Stock{}
	}

	// Helper to safely get string
	getString := func(fields map[string]interface{}, key string) string {
		if val, ok := fields[key].(string); ok {
			return val
		}
		return ""
	}

	// Helper to safely get float64
	getFloat := func(fields map[string]interface{}, key string) float64 {
		if val, ok := fields[key].(float64); ok {
			return val
		}
		return 0.0
	}

	// Build results with scores
	type ScoredStock struct {
		Stock      models.Stock
		TextScore  float64
		FinalScore float64
	}

	var scoredResults []ScoredStock
	for _, hit := range searchResults.Hits {
		popularityScore := getFloat(hit.Fields, "popularity_score")

		stock := models.Stock{
			Symbol:          getString(hit.Fields, "symbol"),
			Name:            getString(hit.Fields, "name"),
			Exchange:        getString(hit.Fields, "exchange"),
			Type:            getString(hit.Fields, "type"),
			Brand:           getString(hit.Fields, "brand"),
			Sector:          getString(hit.Fields, "sector"),
			Industry:        getString(hit.Fields, "industry"),
			Tags:            getString(hit.Fields, "tags"),
			PopularityScore: popularityScore,
		}

		// Combine text relevance score (from Bleve) with popularity score
		// Formula: final_score = (text_score * 0.7) + (popularity_score * 0.3)
		// This ensures relevance is primary, but popularity provides a boost
		textScore := hit.Score
		finalScore := (textScore * 0.7) + (popularityScore * 0.3)

		scoredResults = append(scoredResults, ScoredStock{
			Stock:      stock,
			TextScore:  textScore,
			FinalScore: finalScore,
		})
	}

	// Sort by final score (descending)
	// Using a simple bubble sort for clarity (could use sort.Slice for production)
	for i := 0; i < len(scoredResults); i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].FinalScore > scoredResults[i].FinalScore {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	// Extract just the stocks
	var results []models.Stock
	for _, scored := range scoredResults {
		results = append(results, scored.Stock)
	}

	return results
}

func (e *BleveEngine) GetBySymbol(symbol string) *models.Stock {
	// Exact match on symbol (lowercase to match index)
	termQuery := bleve.NewTermQuery(strings.ToLower(symbol))
	termQuery.SetField("symbol")

	searchRequest := bleve.NewSearchRequest(termQuery)
	searchRequest.Fields = []string{"symbol", "name", "exchange", "type", "brand", "popularity_score"}
	searchRequest.Size = 1 // Just get one

	searchResults, err := e.index.Search(searchRequest)
	if err != nil || len(searchResults.Hits) == 0 {
		return nil
	}

	hit := searchResults.Hits[0]
	getString := func(fields map[string]interface{}, key string) string {
		if val, ok := fields[key].(string); ok {
			return val
		}
		return ""
	}

	getFloat := func(fields map[string]interface{}, key string) float64 {
		if val, ok := fields[key].(float64); ok {
			return val
		}
		return 0.0
	}

	return &models.Stock{
		Symbol:          getString(hit.Fields, "symbol"),
		Name:            getString(hit.Fields, "name"),
		Exchange:        getString(hit.Fields, "exchange"),
		Type:            getString(hit.Fields, "type"),
		Brand:           getString(hit.Fields, "brand"),
		PopularityScore: getFloat(hit.Fields, "popularity_score"),
	}
}

func (e *BleveEngine) GetStock(symbol, exchange string) *models.Stock {
	// If exchange is provided, search with Symbol AND Exchange
	if exchange != "" {
		symbolQuery := bleve.NewTermQuery(strings.ToLower(symbol))
		symbolQuery.SetField("symbol")

		exchangeQuery := bleve.NewTermQuery(strings.ToLower(exchange))
		exchangeQuery.SetField("exchange")

		query := bleve.NewConjunctionQuery(symbolQuery, exchangeQuery)

		searchRequest := bleve.NewSearchRequest(query)
		searchRequest.Fields = []string{"symbol", "name", "exchange", "type", "brand", "popularity_score"}
		searchRequest.Size = 1

		searchResults, err := e.index.Search(searchRequest)
		if err == nil && len(searchResults.Hits) > 0 {
			hit := searchResults.Hits[0]
			getString := func(fields map[string]interface{}, key string) string {
				if val, ok := fields[key].(string); ok {
					return val
				}
				return ""
			}
			getFloat := func(fields map[string]interface{}, key string) float64 {
				if val, ok := fields[key].(float64); ok {
					return val
				}
				return 0.0
			}
			return &models.Stock{
				Symbol:          getString(hit.Fields, "symbol"),
				Name:            getString(hit.Fields, "name"),
				Exchange:        getString(hit.Fields, "exchange"),
				Type:            getString(hit.Fields, "type"),
				Brand:           getString(hit.Fields, "brand"),
				PopularityScore: getFloat(hit.Fields, "popularity_score"),
			}
		}
	}

	// Fallback to GetBySymbol if exchange is empty or not found
	return e.GetBySymbol(symbol)
}

func (e *BleveEngine) Close() error {
	return e.index.Close()
}
