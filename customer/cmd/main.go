package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

// Customer represents customer data
type Customer struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
	Contact string `json:"contact"`
}

// Map to store customer data
var customerData = map[string]Customer{
	"CUSTOMER-0001": {
		Name:    "Samantha Johnson",
		Email:   "samantha.johnson@example.com",
		Address: "123 Main Street, Anytown, USA",
		Contact: "+1 (555) 123-4567",
	},
	"CUSTOMER-0002": {
		Name:    "Michael Thompson",
		Email:   "michael.thompson@example.com",
		Address: "456 Elm Street, Anycity, USA",
		Contact: "+1 (555) 987-6543",
	},
	"CUSTOMER-0003": {
		Name:    "Emily Rodriguez",
		Email:   "emily.rodriguez@example.com",
		Address: "789 Oak Avenue, Anyville, USA",
		Contact: "+1 (555) 321-7890",
	},
	"CUSTOMER-0004": {
		Name:    "David Lee",
		Email:   "david.lee@example.com",
		Address: "101 Pine Road, Anystate, USA",
		Contact: "+1 (555) 876-5432",
	},
}

func main() {
	r := chi.NewRouter()

	r.Get("/api/customers/{customerID}", func(w http.ResponseWriter, r *http.Request) {
		customerID := chi.URLParam(r, "customerID")

		customer, ok := customerData[customerID]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(customer)
	})

	fmt.Println("Server is running on :" + os.Getenv("PORT"))
	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}
