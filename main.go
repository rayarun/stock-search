package main

import (
	"fmt"
	"log"
	"net/http"
	"stock-search/api"
	"stock-search/loader"
	"stock-search/search"
)

func main() {
	// Load environment variables (if any)
	// Load NSE Equity Data (Bulk)
	nseStocks, err := loader.LoadNSEStocks("data/nse_equity.csv")
	if err != nil {
		log.Printf("Warning: Failed to load NSE equity data: %v", err)
	}
	fmt.Printf("Loaded %d NSE stocks.\n", len(nseStocks))

	// Load BSE Equity Data (Bulk)
	bseStocks, err := loader.LoadBSEStocks("data/bse_equity.csv")
	if err != nil {
		log.Printf("Warning: Failed to load BSE equity data: %v", err)
	}
	fmt.Printf("Loaded %d BSE stocks.\n", len(bseStocks))

	// Load Curated Stocks (with Brand data)
	curatedStocks, err := loader.LoadStocks("data/stocks.csv")
	if err != nil {
		log.Fatalf("Failed to load curated stocks: %v", err)
	}
	fmt.Printf("Loaded %d curated stocks.\n", len(curatedStocks))

	// Merge stocks (Curated should come last to overwrite duplicates in index)
	allStocks := append(nseStocks, bseStocks...)
	allStocks = append(allStocks, curatedStocks...)
	fmt.Printf("Total stocks to index: %d\n", len(allStocks))

	// Load Brand Mappings
	brandMappings, err := loader.LoadBrandMappings("data/brand_mappings.json")
	if err != nil {
		log.Printf("Warning: Failed to load brand mappings: %v", err)
	} else {
		fmt.Printf("Loaded %d brand mappings.\n", len(brandMappings))
		// Enrich stocks with brands
		for i := range allStocks {
			if brands, ok := brandMappings[allStocks[i].Symbol]; ok {
				if allStocks[i].Brand != "" {
					allStocks[i].Brand += ", " + brands
				} else {
					allStocks[i].Brand = brands
				}
			}
		}
	}

	// Initialize Search Engine (Bleve)
	engine, err := search.NewBleveEngine("stock_index.bleve", allStocks)
	if err != nil {
		log.Fatalf("Failed to initialize search engine: %v", err)
	}
	defer engine.Close()

	// Initialize API handler
	handler := api.NewHandler(engine)

	// Setup routes
	http.HandleFunc("/search", handler.Search)
	http.HandleFunc("/api/stock", handler.GetStock)

	// Serve static files with no-cache headers for development
	fs := http.FileServer(http.Dir("./static"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fs.ServeHTTP(w, r)
	})

	// Start server
	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
