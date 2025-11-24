package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock-search/credentials"
	"time"
)

// AngelOneConfig holds the configuration for Angel One API
type AngelOneConfig struct {
	CredProvider credentials.Provider
	BaseURL      string
}

// AngelOneClient handles Angel One API interactions
type AngelOneClient struct {
	config    *AngelOneConfig
	jwtToken  string
	tokenTime time.Time
}

// Angel One API structures
type AngelOneLoginRequest struct {
	ClientCode string `json:"clientcode"`
	Password   string `json:"password"`
}

type AngelOneLoginResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		JWTToken     string `json:"jwtToken"`
		RefreshToken string `json:"refreshToken"`
		FeedToken    string `json:"feedToken"`
	} `json:"data"`
}

type AngelOneCandleRequest struct {
	Exchange    string `json:"exchange"`
	SymbolToken string `json:"symboltoken"`
	Interval    string `json:"interval"`
	FromDate    string `json:"fromdate"`
	ToDate      string `json:"todate"`
}

type AngelOneCandleResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    [][]interface{} `json:"data"` // [timestamp, open, high, low, close, volume]
}

// NewAngelOneClient creates a new Angel One API client
func NewAngelOneClient(credProvider credentials.Provider) *AngelOneClient {
	return &AngelOneClient{
		config: &AngelOneConfig{
			CredProvider: credProvider,
			BaseURL:      "https://apiconnect.angelbroking.com",
		},
	}
}

// Authenticate performs login and retrieves JWT token
func (c *AngelOneClient) Authenticate() error {
	// Check if token is still valid (valid for 10 minutes)
	if c.jwtToken != "" && time.Since(c.tokenTime) < 9*time.Minute {
		return nil
	}

	clientCode, err := c.config.CredProvider.GetCredential("ANGELONE_CLIENT_CODE")
	if err != nil {
		return fmt.Errorf("failed to get client code: %v", err)
	}

	password, err := c.config.CredProvider.GetCredential("ANGELONE_PASSWORD")
	if err != nil {
		return fmt.Errorf("failed to get password: %v", err)
	}

	apiKey, err := c.config.CredProvider.GetCredential("ANGELONE_API_KEY")
	if err != nil {
		return fmt.Errorf("failed to get API key: %v", err)
	}

	loginReq := AngelOneLoginRequest{
		ClientCode: clientCode,
		Password:   password,
	}

	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %v", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL+"/rest/auth/angelbroking/user/v1/loginByPassword", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-UserType", "USER")
	req.Header.Set("X-SourceID", "WEB")
	req.Header.Set("X-ClientLocalIP", "127.0.0.1")
	req.Header.Set("X-ClientPublicIP", "127.0.0.1")
	req.Header.Set("X-MACAddress", "00:00:00:00:00:00")
	req.Header.Set("X-PrivateKey", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	var loginResp AngelOneLoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !loginResp.Status {
		return fmt.Errorf("authentication failed: %s", loginResp.Message)
	}

	c.jwtToken = loginResp.Data.JWTToken
	c.tokenTime = time.Now()

	return nil
}

// GetHistoricalData fetches historical candle data
func (c *AngelOneClient) GetHistoricalData(exchange, symbolToken, interval, fromDate, toDate string) ([]PricePoint, error) {
	if err := c.Authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	apiKey, _ := c.config.CredProvider.GetCredential("ANGELONE_API_KEY")

	candleReq := AngelOneCandleRequest{
		Exchange:    exchange,
		SymbolToken: symbolToken,
		Interval:    interval,
		FromDate:    fromDate,
		ToDate:      toDate,
	}

	reqBody, err := json.Marshal(candleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL+"/rest/secure/angelbroking/historical/v1/getCandleData", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-UserType", "USER")
	req.Header.Set("X-SourceID", "WEB")
	req.Header.Set("X-ClientLocalIP", "127.0.0.1")
	req.Header.Set("X-ClientPublicIP", "127.0.0.1")
	req.Header.Set("X-MACAddress", "00:00:00:00:00:00")
	req.Header.Set("X-PrivateKey", apiKey)
	req.Header.Set("Authorization", "Bearer "+c.jwtToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var candleResp AngelOneCandleResponse
	if err := json.Unmarshal(body, &candleResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !candleResp.Status {
		return nil, fmt.Errorf("API error: %s", candleResp.Message)
	}

	// Convert Angel One data to our PricePoint format
	var pricePoints []PricePoint
	for _, candle := range candleResp.Data {
		if len(candle) < 5 {
			continue
		}

		// Parse timestamp (format: "2024-01-01T09:15:00+05:30")
		timestamp, ok := candle[0].(string)
		if !ok {
			continue
		}

		// Use close price for our chart
		close, ok := candle[4].(float64)
		if !ok {
			continue
		}

		pricePoints = append(pricePoints, PricePoint{
			Date:  timestamp,
			Price: close,
		})
	}

	return pricePoints, nil
}

// mapPeriodToInterval maps our period to Angel One interval and date range
func mapPeriodToAngelOneParams(period string) (interval string, duration time.Duration) {
	switch period {
	case "1D":
		return "FIVE_MINUTE", 24 * time.Hour
	case "1W":
		return "FIFTEEN_MINUTE", 7 * 24 * time.Hour
	case "1M":
		return "ONE_HOUR", 30 * 24 * time.Hour
	case "6M":
		return "ONE_DAY", 180 * 24 * time.Hour
	case "YTD":
		// Calculate days since Jan 1 of current year
		now := time.Now()
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return "ONE_DAY", time.Since(startOfYear)
	case "1Y":
		return "ONE_DAY", 365 * 24 * time.Hour
	case "5Y":
		return "ONE_WEEK", 5 * 365 * 24 * time.Hour
	default:
		return "ONE_DAY", 365 * 24 * time.Hour
	}
}

// FetchAngelOneData fetches stock data from Angel One API
func FetchAngelOneData(symbol, exchange, period string, credProvider credentials.Provider) (*YahooData, error) {
	client := NewAngelOneClient(credProvider)

	// Map symbol to Angel One token
	// For now, this is a placeholder - in production, you'd query Angel One's master contract API
	// or maintain a mapping database
	symbolToken := getAngelOneToken(symbol, exchange)
	if symbolToken == "" {
		return nil, fmt.Errorf("symbol token not found for %s on %s", symbol, exchange)
	}

	// Map exchange to Angel One exchange code
	angelExchange := "NSE"
	if exchange == "BSE" {
		angelExchange = "BSE"
	}

	interval, duration := mapPeriodToAngelOneParams(period)
	toDate := time.Now()
	fromDate := toDate.Add(-duration)

	// Format dates as required by Angel One API (YYYY-MM-DD HH:MM)
	fromDateStr := fromDate.Format("2006-01-02 15:04")
	toDateStr := toDate.Format("2006-01-02 15:04")

	history, err := client.GetHistoricalData(angelExchange, symbolToken, interval, fromDateStr, toDateStr)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no data returned from Angel One")
	}

	// Calculate current price and previous close
	currentPrice := history[len(history)-1].Price
	previousDayClose := currentPrice // Placeholder - should be fetched from quote API

	return &YahooData{
		CurrentPrice:     currentPrice,
		PreviousDayClose: previousDayClose,
		History:          history,
	}, nil
}

// getAngelOneToken returns the Angel One symbol token for a given symbol
// This is a placeholder - in production, implement proper token lookup
func getAngelOneToken(symbol, exchange string) string {
	// Common stock tokens (these are examples - use actual tokens from Angel One)
	tokens := map[string]map[string]string{
		"NSE": {
			"RELIANCE":   "2885",
			"TCS":        "11536",
			"HDFCBANK":   "1333",
			"INFY":       "1594",
			"ICICIBANK":  "4963",
			"SBIN":       "3045",
			"BHARTIARTL": "10604",
			"ITC":        "1660",
			"KOTAKBANK":  "1922",
			"LT":         "11483",
		},
		"BSE": {
			"RELIANCE":  "500325",
			"TCS":       "532540",
			"HDFCBANK":  "500180",
			"INFY":      "500209",
			"ICICIBANK": "532174",
		},
	}

	if exchangeTokens, ok := tokens[exchange]; ok {
		if token, ok := exchangeTokens[symbol]; ok {
			return token
		}
	}

	return ""
}
