package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

// Account represents account data
type Account struct {
	ProductDescription string  `json:"productDescription"`
	Quantity           int     `json:"quantity"`
	UnitPrice          float64 `json:"unitPrice"`
	Price              float64 `json:"price"`
	SubTotal           float64 `json:"subTotal"`
	Tax                int     `json:"tax"`
	TaxAmount          float64 `json:"taxAmount"`
	GrandTotal         float64 `json:"grandTotal"`
	Currency           string  `json:"currency"`
	CurrencySymbol     string  `json:"currencySymbol"`
}

// Map to store account data
var accountData = map[string]map[string]Account{
	"CUSTOMER-0001": {
		"PRD-160": {
			ProductDescription: "Product 1",
			Quantity:           1,
			UnitPrice:          103.00,
			Price:              103.00,
			SubTotal:           103.00,
			Tax:                10,
			TaxAmount:          10.3,
			GrandTotal:         113.3,
			Currency:           "EUR",
			CurrencySymbol:     "€",
		},
	},
	"CUSTOMER-0002": {
		"PRD-160": {
			ProductDescription: "Product 1",
			Quantity:           1,
			UnitPrice:          103.00,
			Price:              103.00,
			SubTotal:           103.00,
			Tax:                10,
			TaxAmount:          10.3,
			GrandTotal:         113.3,
			Currency:           "EUR",
			CurrencySymbol:     "€",
		},
	},
	"CUSTOMER-0003": {
		"PRD-400": {
			ProductDescription: "Product 2",
			Quantity:           2,
			UnitPrice:          10.50,
			Price:              21.00,
			SubTotal:           21.00,
			Tax:                5,
			TaxAmount:          1.05,
			GrandTotal:         22.05,
			Currency:           "USD",
			CurrencySymbol:     "$",
		},
	},
	"CUSTOMER-0004": {
		"PRD-799": {
			ProductDescription: "Product 3",
			Quantity:           1,
			UnitPrice:          10.50,
			Price:              10.50,
			SubTotal:           10.50,
			Tax:                10,
			TaxAmount:          1.05,
			GrandTotal:         11.05,
			Currency:           "GBP",
			CurrencySymbol:     "£",
		},
	},
}

func main() {
	r := chi.NewRouter()

	r.Get("/api/accounts/{customerID}/{productID}", func(w http.ResponseWriter, r *http.Request) {
		customerID := chi.URLParam(r, "customerID")
		productID := chi.URLParam(r, "productID")

		account, ok := accountData[customerID][productID]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	})

	fmt.Println("Server is running on :" + os.Getenv("PORT"))
	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}
