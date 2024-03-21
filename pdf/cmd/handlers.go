package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// GenerateInvoicePDFHandler handles the request for generating an invoice PDF
func GenerateInvoicePDFHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	var invoice Invoice
	if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	if err := validateInvoice(&invoice); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call the processInvoice function in a goroutine
	go func(invoice Invoice) {
		mutex.Lock()
		defer mutex.Unlock()

		if err := processInvoice(&invoice); err != nil {
			log.Printf("error processing invoice: %v\n", err)
		}
	}(invoice)

	// Return success response
	w.WriteHeader(http.StatusOK)
}

func processInvoice(invoice *Invoice) error {
	existingInvoice, err := getPdfInvoiceByInvoiceID(invoice.InvoiceID)
	if err != nil {
		return fmt.Errorf("error checking existing record: %v", err)
	}

	if existingInvoice != nil {
		// Update existing invoice record
		invoice.ID = existingInvoice.ID
		if err := updateInvoice(*invoice); err != nil {
			return fmt.Errorf("failed to update invoice in the database: %v", err)
		}
	} else {
		// Insert new invoice record
		if err := insertInvoice(invoice); err != nil {
			return fmt.Errorf("failed to save invoice to database: %v", err)
		}
	}

	if err := generateAndSendInvoicePDF(*invoice); err != nil {
		return fmt.Errorf("failed to generate and send PDF: %v", err)
	}

	return nil
}

func generateAndSendInvoicePDF(invoice Invoice) error {
	var b bytes.Buffer
	if err := generatePDF(
		invoice, &b,
		os.Getenv("COMPANY_NO"), os.Getenv("COMPANY_NAME"),
		os.Getenv("COMPANY_ADDRESS"), os.Getenv("COMPANY_CONTACT"),
		os.Getenv("COMPANY_LOGO_PATH"), os.Getenv("COMPANY_LOGO_IMG_TYPE"),
	); err != nil {
		return fmt.Errorf("failed to generate PDF: %v", err)
	}

	if err := createInvoiceAndSendAPIRequest(
		b, invoice.ProductCode, invoice.CustomerID, invoice.InvoiceID, invoice.EmailTo, getDoneURL(invoice)); err != nil {
		return fmt.Errorf("failed to call email service: %v", err)
	}

	return nil
}

func validateInvoice(inv *Invoice) error {
	if inv == nil {
		return errors.New("invoice is nil")
	}

	if inv.ProductCode == "" {
		return errors.New("empty product code")
	}

	if inv.CustomerID == "" {
		return errors.New("empty customer ID")
	}

	if inv.InvoiceID == "" {
		return errors.New("empty invoice ID")
	}

	if inv.EmailTo == "" {
		return errors.New("empty email to")
	}

	if inv.InvoiceDate == "" {
		return errors.New("empty invoice date")
	}

	if inv.Name == "" {
		return errors.New("empty name")
	}

	if inv.Address == "" {
		return errors.New("empty address")
	}

	if inv.Contact == "" {
		return errors.New("empty contact")
	}

	if inv.Tax < 0 {
		return errors.New("invalid tax")
	}

	if inv.Unit <= 0 {
		return errors.New("invalid unit")
	}

	if inv.Currency == "" {
		return errors.New("empty currency")
	}

	if inv.CurrencySymbol == "" {
		return errors.New("empty currency symbol")
	}

	if inv.DoneURL == "" {
		return errors.New("empty done URL")
	}

	return nil
}

// CBInvoicePdfHandler handles the callback request from email service
func CBInvoicePdfHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the invoice ID from the route parameters
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	invoice, err := getPdfInvoiceByID(id)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if invoice == nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	// Parse request body
	var requestBody struct {
		Status         int16   `json:"status"`
		FailedMessage  *string `json:"failedMessage,omitempty"`
		FailedAt       *string `json:"failedAt,omitempty"`
		SuccessMessage *string `json:"successMessage,omitempty"`
		InvoiceSentAt  *string `json:"invoiceSentAt,omitempty"`
		ID             *int    `json:"ID,omitempty"`
	}

	err = json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	emailServiceMessage, emailServiceTriggeredAt := requestBody.FailedMessage, requestBody.FailedAt
	if requestBody.Status == http.StatusOK {
		emailServiceMessage, emailServiceTriggeredAt = requestBody.SuccessMessage, requestBody.InvoiceSentAt
	}
	err = updateEmailServiceFieldsByID(invoice.ID, requestBody.ID,
		*emailServiceMessage, requestBody.Status, *emailServiceTriggeredAt)
	if err != nil {
		log.Printf("Failed to update invoice in the database: %v\n", err)
		http.Error(w, "Failed to update invoice in the database", http.StatusInternalServerError)
		return
	}

	reqBody := struct {
		ID                      int    `json:"id"`
		EmailServiceID          *int   `json:"emailServiceID,omitempty"`
		EmailServiceMessage     string `json:"emailServiceMessage"`
		EmailServiceStatus      int16  `json:"emailServiceStatus"`
		EmailServiceTriggeredAt string `json:"emailServiceTriggeredAt"`
	}{
		ID:                      invoice.ID,
		EmailServiceID:          requestBody.ID,
		EmailServiceMessage:     *emailServiceMessage,
		EmailServiceStatus:      requestBody.Status,
		EmailServiceTriggeredAt: *emailServiceTriggeredAt,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	// Send POST request to the invoice DoneURL
	resp, err := http.Post(invoice.DoneURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Failed to send POST request to DoneURL: %v\n", err)
		http.Error(w, "Failed to send POST request to DoneURL", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code: %d\n", resp.StatusCode)
		http.Error(w, "Unexpected status code", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
}

func InvoicePDFByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	invoiceID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	invoice, err := getPdfInvoiceByID(invoiceID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if invoice == nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	if err := generateAndSendInvoicePDF(*invoice); err != nil {
		log.Printf("Failed to generate and send PDF: %v\n", err)
		http.Error(w, "Failed to generate and send PDF", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
}
