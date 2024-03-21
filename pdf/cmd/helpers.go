package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/arifmahmudrana/invoice/pdf"
)

// createInvoiceAndSendAPIRequest creates an invoice and sends API request with specified parameters
func createInvoiceAndSendAPIRequest(b bytes.Buffer, productCode, customerID, invoiceID, emailTo, doneURL string) error {
	// Calculate hash of buffer
	fileHash := calculateSHA1Hash(b.Bytes())

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Write buffer as a multipart file
	part, err := writer.CreateFormFile("invoiceFile", "invoice.pdf")
	if err != nil {
		return err
	}
	if _, err := part.Write(b.Bytes()); err != nil {
		return err
	}

	// Write other fields
	writer.WriteField("productCode", productCode)
	writer.WriteField("customerID", customerID)
	writer.WriteField("invoiceID", invoiceID)
	writer.WriteField("emailTo", emailTo)
	writer.WriteField("doneURL", doneURL)
	writer.WriteField("fileHash", fileHash)

	// Close multipart writer
	if err := writer.Close(); err != nil {
		return err
	}

	// Make POST request to API
	resp, err := http.Post(os.Getenv("EMAIL_SVC"), writer.FormDataContentType(), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status code %d", resp.StatusCode)
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Handle response as needed
	fmt.Println(string(responseBody))

	return nil
}

// calculateSHA1Hash calculates the SHA-1 hash of data and returns it as a string
func calculateSHA1Hash(data []byte) string {
	hasher := sha1.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

func generatePDF(
	invoice Invoice, w io.Writer,
	comNo, frName, frAdd, frCon,
	logo, logoType string) error {
	ig := pdf.NewInvoiceGenerator()
	ig.SetInvoiceNo(invoice.InvoiceID)
	ig.SetInvoiceDate(invoice.InvoiceDate)
	ig.SetCompanyNo(comNo)
	ig.SetFromName(frName)
	ig.SetFromAddress(frAdd)
	ig.SetFromContact(frCon)
	ig.SetToName(invoice.Name)
	ig.SetToAddress(invoice.Address)
	ig.SetToContact(invoice.Contact)

	// data := [][]string{
	// 	{invoice.Unit, invoice.Description, invoice.PricePerUnit},
	// }

	if err := ig.GenerateInvoice(pdf.SubscriptionInfo{
		ProductDescription: invoice.Description,
		Quantity:           invoice.Unit,
		UnitPrice:          invoice.PricePerUnit,
		Price:              invoice.Price,
		SubTotal:           invoice.SubTotal,
		Tax:                invoice.Tax,
		TaxAmount:          invoice.TaxAmount,
		GrandTotal:         invoice.GrandTotal,
		Currency:           invoice.Currency,
		CurrencySymbol:     invoice.CurrencySymbol,
	}, w, logo, logoType); err != nil {
		log.Printf("Error generating invoice: %v\n", err)
		return err
	}
	return nil
}

func getDoneURL(invoice Invoice) string {
	return fmt.Sprintf("%s%s/%d", os.Getenv("BASE_URL"), cbURLPath, invoice.ID)
}
