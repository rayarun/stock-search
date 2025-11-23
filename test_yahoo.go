package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Yahoo Finance Structures for Test
type YahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
			Timestamp []int64 `json:"timestamp"`
		} `json:"result"`
	} `json:"chart"`
}

func main() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	// 1. Get Cookie from main page
	// We need to look like a real browser
	req, _ := http.NewRequest("GET", "https://finance.yahoo.com", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching main page:", err)
		return
	}
	resp.Body.Close()
	fmt.Println("Main page fetched, cookies set")

	// 2. Get Crumb
	req, _ = http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://finance.yahoo.com")
	req.Header.Set("Referer", "https://finance.yahoo.com/")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("Error fetching crumb:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	crumb := string(body)
	fmt.Println("Crumb fetched:", crumb)

	if strings.Contains(crumb, "html") {
		fmt.Println("Failed to get valid crumb (got HTML)")
		return
	}

	// 3. Get Chart Data
	symbol := "HONASA.NS"
	// Test 1D
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?symbol=%s&range=1d&interval=5m&includePrePost=true&crumb=%s", symbol, symbol, crumb)
	fmt.Println("Fetching URL:", url)

	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("Error fetching chart:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Error status:", resp.Status)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Body:", string(body))
		return
	}

	fmt.Println("Successfully fetched chart data!")

	body, _ = io.ReadAll(resp.Body)

	// Print first 5 and last 5 dates to verify
	var result YahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	if len(result.Chart.Result) > 0 {
		meta := result.Chart.Result[0].Meta
		fmt.Printf("RegularMarketPrice: %f\n", meta.RegularMarketPrice)
		fmt.Printf("RegularMarketTime: %d (%s)\n", meta.RegularMarketTime, time.Unix(meta.RegularMarketTime, 0).Format("2006-01-02 15:04:05"))

		timestamps := result.Chart.Result[0].Timestamp
		fmt.Printf("Total points: %d\n", len(timestamps))
		if len(timestamps) > 0 {
			// Print first 5
			fmt.Println("First 5 points:")
			for i := 0; i < 5 && i < len(timestamps); i++ {
				ts := time.Unix(timestamps[i], 0).Format("2006-01-02 15:04:05")
				fmt.Printf("[%d] %s\n", i, ts)
			}

			// Print last 5
			fmt.Println("Last 5 points:")
			start := len(timestamps) - 5
			if start < 0 {
				start = 0
			}
			for i := start; i < len(timestamps); i++ {
				ts := time.Unix(timestamps[i], 0).Format("2006-01-02 15:04:05")
				fmt.Printf("[%d] %s\n", i, ts)
			}
		}
	} else {
		fmt.Println("No results in chart")
	}
}
