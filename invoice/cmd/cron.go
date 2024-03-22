package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

// processStalledInvoices marks invoices taking longer then 10 minutes as failed
func processStalledInvoices() {
	// clean up failed or stalled invoices
	invoices, err := GetInvoices(db, time.Now().Add(10*time.Minute))
	if err != nil {
		log.Printf("Error calling GetInvoices: %v\n", err)
		return
	}
	// TODO: can we use concurrency
	for _, invoice := range invoices {
		subscription, err := GetSubscriptionByIDCustomerIDProductCode(db, invoice.SubscriptionID, invoice.CustomerID, invoice.ProductCode)
		if err != nil {
			log.Printf("Error GetSubscriptionByIDCustomerIDProductCode: %v\n", err)
			continue
		}

		// Begin the transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error calling Begin for transaction: %v\n", err)
			continue
		}

		if err = SetStatusInvoice(tx, invoice.ID, StatusFailed); err != nil {
			log.Printf("Error calling SetStatusInvoice: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}

		if err = UpdateSubscriptionFields(tx, subscription.ID, subscription.BillingFrequencyRemains, StatusFailed, subscription.NextInvoiceDate); err != nil {
			log.Printf("Error calling UpdateSubscriptionFields: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}

		if err = tx.Commit(); err != nil {
			// Rollback the transaction if commit fails and log the error
			log.Printf("Error calling transaction Commit: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}
	}
	log.Println("Executing hourly task...")
}

// processInvoiceDaily
func processInvoiceDaily() {
	subscriptions, err := GetPendingSubscriptions(db)
	if err != nil {
		log.Printf("Error calling GetPendingSubscriptions: %v\n", err)
		return
	}

	// TODO: can we use concurrency
	for _, subscription := range subscriptions {
		log.Printf("Processing subscription: %#v\n", subscription)
		// Call accounts service for price
		accountsData, err := GetAccountDetails(subscription.CustomerID, subscription.ProductCode)
		if err != nil {
			log.Printf("Error calling GetCustomerDetails: %v\n", err)
			continue
		}

		// Call customer service for customer information
		customerDetails, err := GetCustomerDetails(subscription.CustomerID)
		if err != nil {
			log.Printf("Error calling GetCustomerDetails: %v\n", err)
			continue
		}

		// Create invoice record in DB
		invoicingStartedAt := time.Now().UTC()
		invoiceData := Invoice{
			SubscriptionID:     subscription.ID,
			CustomerID:         subscription.CustomerID,
			ProductCode:        subscription.ProductCode,
			EmailTo:            customerDetails.Email,
			InvoiceDate:        subscription.NextInvoiceDate,
			Name:               customerDetails.Name,
			Address:            customerDetails.Address,
			Contact:            customerDetails.Contact,
			Tax:                accountsData.Tax,
			Unit:               accountsData.Quantity,
			Description:        accountsData.ProductDescription,
			PricePerUnit:       accountsData.UnitPrice,
			Price:              accountsData.Price,
			SubTotal:           accountsData.SubTotal,
			TaxAmount:          accountsData.TaxAmount,
			GrandTotal:         accountsData.GrandTotal,
			Currency:           accountsData.Currency,
			CurrencySymbol:     accountsData.CurrencySymbol,
			InvoicingStartedAt: invoicingStartedAt,
			Status:             StatusProcessing,
		}
		// Begin the transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error calling Begin for transaction: %v\n", err)
			continue
		}
		if err = InsertInvoice(tx, &invoiceData); err != nil {
			log.Printf("Error calling InsertInvoice: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}
		if err = UpdateSubscriptionStatus(tx, invoicingStartedAt, StatusProcessing, subscription.ID); err != nil {
			log.Printf("Error calling UpdateSubscriptionStatus: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}

		// Call PDF service
		reqBody := struct {
			ProductCode    string  `json:"productCode"`
			CustomerID     string  `json:"customerID"`
			InvoiceID      string  `json:"invoiceID"`
			EmailTo        string  `json:"emailTo"`
			InvoiceDate    string  `json:"invoiceDate"`
			Name           string  `json:"name"`
			Address        string  `json:"address"`
			Contact        string  `json:"contact"`
			Tax            int     `json:"tax"`
			Unit           int     `json:"unit"`
			Description    string  `json:"description"`
			PricePerUnit   float64 `json:"pricePerUnit"`
			Price          float64 `json:"price"`
			SubTotal       float64 `json:"subTotal"`
			TaxAmount      float64 `json:"taxAmount"`
			GrandTotal     float64 `json:"grandTotal"`
			Currency       string  `json:"currency"`
			CurrencySymbol string  `json:"currencySymbol"`
			DoneURL        string  `json:"doneURL"`
		}{
			ProductCode:    invoiceData.ProductCode,
			CustomerID:     invoiceData.CustomerID,
			InvoiceID:      invoiceData.GetInvoiceID(),
			EmailTo:        invoiceData.EmailTo,
			InvoiceDate:    invoiceData.InvoiceDate.Format("Jan 02, 2006"),
			Name:           invoiceData.Name,
			Address:        invoiceData.Address,
			Contact:        invoiceData.Contact,
			Tax:            invoiceData.Tax,
			Unit:           invoiceData.Unit,
			Description:    invoiceData.Description,
			PricePerUnit:   invoiceData.PricePerUnit,
			Price:          invoiceData.Price,
			SubTotal:       invoiceData.SubTotal,
			TaxAmount:      invoiceData.TaxAmount,
			GrandTotal:     invoiceData.GrandTotal,
			Currency:       invoiceData.Currency,
			CurrencySymbol: invoiceData.CurrencySymbol,
			DoneURL:        getDoneURL(invoiceData),
		}
		res, err := MakeHTTPRequest(http.MethodPost, os.Getenv("PDF_SVC"), reqBody)
		if err != nil {
			log.Printf("Error calling MakeHTTPRequest: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}
		if err = res.Body.Close(); err != nil {
			log.Printf("Error calling res.Body.Close(): %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}

		if err = tx.Commit(); err != nil {
			// Rollback the transaction if commit fails and log the error
			log.Printf("Error calling transaction Commit: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			continue
		}

		log.Printf("Processed subscription: %#v\n", subscription)
	}

	log.Println("Processing invoicing daily task...")
}
