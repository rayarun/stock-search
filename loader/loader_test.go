package loader

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLoadStocks(t *testing.T) {
	// Create a temporary CSV file
	content := `Symbol,Name,Exchange,Type,Brand
RELIANCE,Reliance Industries Limited,NSE,Stock,Jio
TCS,Tata Consultancy Services Limited,NSE,Stock,Tata`
	tmpfile, err := os.CreateTemp("", "stocks_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test LoadStocks
	stocks, err := LoadStocks(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadStocks failed: %v", err)
	}

	if len(stocks) != 2 {
		t.Errorf("Expected 2 stocks, got %d", len(stocks))
	}

	if stocks[0].Symbol != "RELIANCE" {
		t.Errorf("Expected symbol RELIANCE, got %s", stocks[0].Symbol)
	}
	if stocks[0].Brand != "Jio" {
		t.Errorf("Expected brand Jio, got %s", stocks[0].Brand)
	}
}

func TestLoadBrandMappings(t *testing.T) {
	// Create a temporary JSON file
	content := `{
		"RELIANCE": "Jio, Reliance Digital",
		"ITC": "Aashirvaad"
	}`
	tmpfile, err := os.CreateTemp("", "brands_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test LoadBrandMappings
	mappings, err := LoadBrandMappings(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadBrandMappings failed: %v", err)
	}

	if len(mappings) != 2 {
		t.Errorf("Expected 2 mappings, got %d", len(mappings))
	}

	if val, ok := mappings["RELIANCE"]; !ok || val != "Jio, Reliance Digital" {
		t.Errorf("Incorrect mapping for RELIANCE")
	}
}
