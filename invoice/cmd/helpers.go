package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

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

type Customer struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
	Contact string `json:"contact"`
}

// GetPendingSubscriptions retrieves pending subscriptions from the database.
func GetPendingSubscriptions(db *sql.DB) ([]Subscription, error) {
	currentTime := time.Now()
	return GetSubscriptions(db, currentTime)
}

// MakeHTTPRequest sends a HTTP request with the specified method and optional JSON payload
func MakeHTTPRequest(method, url string, body interface{}) (*http.Response, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Prepare the request body if provided
	var reqBody []byte
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = jsonBody
	}

	// Create the request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	// Set the request headers for other then GET requests with body
	if method != http.MethodGet && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}

// GetAccountDetails retrieves account details from the accounts service.
func GetAccountDetails(customerID, productID string) (*Account, error) {
	accountsURL := fmt.Sprintf("%s/api/accounts/%s/%s", os.Getenv("ACCOUNTS_SERVICE_BASE_URL"), customerID, productID)
	resp, err := MakeHTTPRequest("GET", accountsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to accounts service: %v", err)
	}
	defer resp.Body.Close()
	return parseAccountResponse(resp.Body)
}

// GetCustomerDetails retrieves customer details from the customer service.
func GetCustomerDetails(customerID string) (*Customer, error) {
	customerURL := fmt.Sprintf("%s/api/customers/%s", os.Getenv("CUSTOMER_SERVICE_BASE_URL"), customerID)
	resp, err := MakeHTTPRequest("GET", customerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to customer service: %v", err)
	}
	defer resp.Body.Close()
	return parseCustomerResponse(resp.Body)
}

func parseAccountResponse(body io.Reader) (*Account, error) {
	var account Account
	if err := json.NewDecoder(body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}
	return &account, nil
}

func parseCustomerResponse(body io.Reader) (*Customer, error) {
	var customer Customer
	if err := json.NewDecoder(body).Decode(&customer); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}
	return &customer, nil
}

func getDoneURL(invoice Invoice) string {
	return fmt.Sprintf("%s%s/%s", os.Getenv("BASE_URL"), cbURLPath, invoice.GetInvoiceID())
}

func getNextInvoiceDate(subscription Subscription) (time.Time, error) {
	switch subscription.BillingFrequencyUnits {
	case "MONTHS":
		return subscription.NextInvoiceDate.AddDate(0, subscription.Duration/subscription.BillingFrequency, 0), nil
	}

	return time.Time{}, fmt.Errorf("error unknown BillingFrequencyUnits: %s", subscription.BillingFrequencyUnits)
}
