package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"stock-search/credentials"
	"stock-search/models"
	"stock-search/search"
	"strings"
	"time"
)

type Handler struct {
	Engine search.SearchEngine
}

func NewHandler(engine search.SearchEngine) *Handler {
	return &Handler{Engine: engine}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	results := h.Engine.Search(query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "Missing symbol parameter", http.StatusBadRequest)
		return
	}

	// Get period parameter (default to 1D if not provided)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1D"
	}

	// Get exchange parameter (optional)
	exchange := r.URL.Query().Get("exchange")

	var stock *models.Stock
	if exchange != "" {
		stock = h.Engine.GetStock(symbol, exchange)
	} else {
		stock = h.Engine.GetBySymbol(symbol)
	}

	if stock == nil {
		http.Error(w, "Stock not found", http.StatusNotFound)
		return
	}

	// Get data provider parameter (default to yahoo)
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "yahoo"
	}

	var stockData *YahooData
	var err error

	// Try Angel One if selected
	if provider == "angelone" {
		// Use environment variable provider by default
		// Users can replace this with KMS provider
		credProvider := credentials.NewEnvProvider()
		stockData, err = FetchAngelOneData(stock.Symbol, stock.Exchange, period, credProvider)

		if err != nil {
			// Fallback to Yahoo Finance
			fmt.Printf("Angel One failed (%v), falling back to Yahoo Finance\n", err)
			stockData, err = fetchYahooData(stock.Symbol, stock.Exchange, period)
		}
	} else {
		// Use Yahoo Finance
		stockData, err = fetchYahooData(stock.Symbol, stock.Exchange, period)
	}

	if err != nil {
		// Fallback to mock data if both providers fail
		fmt.Println("Error fetching data:", err)
		serveMockData(w, stock, period)
		return
	}

	response := struct {
		*models.Stock
		CurrentPrice     float64      `json:"currentPrice"`
		PreviousDayClose float64      `json:"previousDayClose"` // Closing price of day before chart starts
		History          []PricePoint `json:"history"`
	}{
		Stock:            stock,
		CurrentPrice:     stockData.CurrentPrice,
		PreviousDayClose: stockData.PreviousDayClose,
		History:          stockData.History,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func serveMockData(w http.ResponseWriter, stock *models.Stock, period string) {
	// Mock Price Data (Fallback)
	currentPrice := 1000.0 + (float64(len(stock.Symbol)) * 10.5)

	var history []PricePoint
	basePrice := currentPrice * 0.9

	// Generate mock data based on period granularity
	switch period {
	case "1D":
		// 5-minute intervals for 1 day (6.5 hours of trading)
		for i := 78; i >= 0; i-- {
			timestamp := time.Now().Add(-time.Duration(i) * 5 * time.Minute)
			fluctuation := float64(i%5) * 2.0
			if i%2 == 0 {
				fluctuation = -fluctuation
			}
			price := basePrice + fluctuation + (float64(i) * 0.1)
			history = append(history, PricePoint{Date: timestamp.Format(time.RFC3339), Price: price})
		}
	case "1W", "1M":
		// Hourly intervals
		hours := 168 // 1 week
		if period == "1M" {
			hours = 720 // 30 days
		}
		for i := hours; i >= 0; i-- {
			timestamp := time.Now().Add(-time.Duration(i) * time.Hour)
			fluctuation := float64(i%10) * 1.5
			if i%2 == 0 {
				fluctuation = -fluctuation
			}
			price := basePrice + fluctuation + (float64(i) * 0.05)
			history = append(history, PricePoint{Date: timestamp.Format(time.RFC3339), Price: price})
		}
	default:
		// Daily intervals for 6M, 1Y, 5Y
		days := 30
		if period == "6M" {
			days = 180
		} else if period == "1Y" {
			days = 365
		} else if period == "YTD" {
			days = int(time.Since(time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)).Hours() / 24)
		} else if period == "5Y" {
			days = 1825
		}
		for i := days; i >= 0; i-- {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			fluctuation := float64(i%5) * 2.0
			if i%2 == 0 {
				fluctuation = -fluctuation
			}
			price := basePrice + fluctuation + (float64(i) * 0.5)
			history = append(history, PricePoint{Date: date, Price: price})
		}
	}

	response := struct {
		*models.Stock
		CurrentPrice     float64      `json:"currentPrice"`
		PreviousDayClose float64      `json:"previousDayClose"`
		History          []PricePoint `json:"history"`
	}{
		Stock:            stock,
		CurrentPrice:     currentPrice,
		PreviousDayClose: basePrice, // Mock previous close as basePrice
		History:          history,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Yahoo Finance Structures
type YahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

type PricePoint struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}

type YahooData struct {
	CurrentPrice     float64
	PreviousDayClose float64 // Closing price of day before chart starts
	History          []PricePoint
}

func fetchYahooData(symbol string, exchange string, period string) (*YahooData, error) {
	// Map period to Yahoo Finance parameters
	var yahooRange, yahooInterval string
	switch period {
	case "1D":
		yahooRange = "1d"
		yahooInterval = "5m"
	case "1W":
		yahooRange = "7d"
		yahooInterval = "5m" // Changed from 60m to 5m
	case "1M":
		yahooRange = "1mo"
		yahooInterval = "30m" // Changed from 60m to 30m
	case "6M":
		yahooRange = "6mo"
		yahooInterval = "1d"
	case "1Y":
		yahooRange = "1y"
		yahooInterval = "1d"
	case "YTD":
		yahooRange = "ytd"
		yahooInterval = "1d"
	case "5Y":
		yahooRange = "5y"
		yahooInterval = "1wk" // Changed from 1d to 1wk (weekly)
	default:
		yahooRange = "1d"
		yahooInterval = "5m"
	}

	// Yahoo Finance requires exchange-specific suffix
	// .NS for NSE, .BO for BSE
	var suffix string
	if exchange == "BSE" {
		suffix = ".BO"
	} else {
		suffix = ".NS" // Default to NSE
	}

	// URL encode the symbol to handle special characters like '&' (e.g. M&M)
	yahooSymbol := url.QueryEscape(symbol + suffix)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	// 1. Get Cookie from main page
	req, _ := http.NewRequest("GET", "https://finance.yahoo.com", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching cookie for %s: %v\n", symbol, err)
		return nil, fmt.Errorf("failed to get cookie: %v", err)
	}
	resp.Body.Close()

	// 2. Get Crumb
	req, _ = http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://finance.yahoo.com")
	req.Header.Set("Referer", "https://finance.yahoo.com/")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching crumb for %s: %v\n", symbol, err)
		return nil, fmt.Errorf("failed to get crumb: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	crumb := string(body)

	if strings.Contains(crumb, "html") {
		fmt.Printf("Invalid crumb for %s: %s\n", symbol, crumb)
		return nil, fmt.Errorf("invalid crumb received")
	}

	// 3. Get Chart Data with dynamic range and interval
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?symbol=%s&range=%s&interval=%s&crumb=%s",
		yahooSymbol, yahooSymbol, yahooRange, yahooInterval, crumb)

	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching chart for %s: %v\n", symbol, err)
		return nil, fmt.Errorf("failed to fetch chart: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Yahoo API error for %s: Status %s\n", symbol, resp.Status)
		return nil, fmt.Errorf("yahoo api returned status: %s", resp.Status)
	}

	var yahooResp YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&yahooResp); err != nil {
		fmt.Printf("JSON decode error for %s: %v\n", symbol, err)
		return nil, fmt.Errorf("failed to decode json: %v", err)
	}

	if len(yahooResp.Chart.Result) == 0 {
		fmt.Printf("No result in Yahoo response for %s\n", symbol)
		return nil, fmt.Errorf("no result in yahoo response")
	}

	result := yahooResp.Chart.Result[0]
	currentPrice := result.Meta.RegularMarketPrice
	previousDayClose := result.Meta.ChartPreviousClose

	var history []PricePoint
	timestamps := result.Timestamp
	closes := result.Indicators.Quote[0].Close

	// Format timestamps based on interval (intraday vs daily/weekly)
	isIntraday := yahooInterval == "5m" || yahooInterval == "30m" || yahooInterval == "60m"

	for i, ts := range timestamps {
		if i < len(closes) && closes[i] != 0 {
			var dateStr string
			if isIntraday {
				// For intraday data, include time in RFC3339 format
				dateStr = time.Unix(ts, 0).Format(time.RFC3339)
			} else {
				// For daily data, use date only
				dateStr = time.Unix(ts, 0).Format("2006-01-02")
			}
			history = append(history, PricePoint{Date: dateStr, Price: closes[i]})
		}
	}

	// Append closing price if missing (for 1D/intraday)
	if isIntraday && len(history) > 0 {
		lastPoint := history[len(history)-1]
		lastTime, _ := time.Parse(time.RFC3339, lastPoint.Date)
		regularTime := time.Unix(result.Meta.RegularMarketTime, 0)

		// If last point is more than 1 minute before regular market time, append regular market price
		if regularTime.Sub(lastTime) > 1*time.Minute {
			history = append(history, PricePoint{
				Date:  regularTime.Format(time.RFC3339),
				Price: result.Meta.RegularMarketPrice,
			})
		}
	}

	// If PreviousDayClose is 0 (e.g. some Yahoo responses might miss it), fallback to first history point
	if previousDayClose == 0 && len(history) > 0 {
		previousDayClose = history[0].Price
	}

	return &YahooData{
		CurrentPrice:     currentPrice,
		PreviousDayClose: previousDayClose,
		History:          history,
	}, nil
}
