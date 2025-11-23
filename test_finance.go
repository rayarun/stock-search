package main

import (
	"fmt"
	"github.com/piquette/finance-go/quote"
)

func main() {
	q, err := quote.Get("RELIANCE.NS")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if q == nil {
		fmt.Println("Quote is nil")
		return
	}
	fmt.Printf("Symbol: %s, Price: %f\n", q.Symbol, q.RegularMarketPrice)
}
