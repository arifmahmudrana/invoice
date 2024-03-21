package main

import (
	"database/sql"
	"fmt"
	"time"
)

var db *sql.DB

type Invoice struct {
	ID                      int            `json:"id"`
	ProductCode             string         `json:"productCode"`
	CustomerID              string         `json:"customerID"`
	InvoiceID               string         `json:"invoiceID"`
	EmailTo                 string         `json:"emailTo"`
	InvoiceDate             string         `json:"invoiceDate"`
	Name                    string         `json:"name"`
	Address                 string         `json:"address"`
	Contact                 string         `json:"contact"`
	Tax                     int            `json:"tax"`
	Unit                    int            `json:"unit"`
	Description             string         `json:"description"`
	PricePerUnit            float64        `json:"pricePerUnit"`
	Price                   float64        `json:"price"`
	SubTotal                float64        `json:"subTotal"`
	TaxAmount               float64        `json:"taxAmount"`
	GrandTotal              float64        `json:"grandTotal"`
	Currency                string         `json:"currency"`
	CurrencySymbol          string         `json:"currencySymbol"`
	DoneURL                 string         `json:"doneURL"`
	EmailServiceID          sql.NullInt64  `json:"emailServiceID,omitempty"`
	EmailServiceMessage     sql.NullString `json:"emailServiceMessage,omitempty"`
	EmailServiceStatus      sql.NullInt16  `json:"emailServiceStatus,omitempty"`
	EmailServiceTriggeredAt *time.Time     `json:"emailServiceTriggeredAt,omitempty"`
}

func createTable() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pdf_invoices (
        id INT AUTO_INCREMENT PRIMARY KEY,
        product_code VARCHAR(255) NOT NULL,
        customer_id VARCHAR(255) NOT NULL,
        invoice_id VARCHAR(255) NOT NULL,
        email_to VARCHAR(255) NOT NULL,
        invoice_date VARCHAR(255) NOT NULL,
        name VARCHAR(255) NOT NULL,
        address VARCHAR(255) NOT NULL,
        contact VARCHAR(255) NOT NULL,
        tax INT NOT NULL,
        unit INT NOT NULL,
        description VARCHAR(255) NOT NULL,
        price_per_unit DECIMAL(10, 2),
				price DECIMAL(10, 2) NOT NULL,
				sub_total DECIMAL(10, 2) NOT NULL,
				tax_amount DECIMAL(10, 2) NOT NULL,
				grand_total DECIMAL(10, 2) NOT NULL,
				currency VARCHAR(3) NOT NULL,
				currency_symbol VARCHAR(5) NOT NULL,
        done_url VARCHAR(255) NOT NULL,
        email_service_id INT DEFAULT NULL,
        email_service_message VARCHAR(255) DEFAULT NULL,
        email_service_status SMALLINT UNSIGNED DEFAULT NULL,
        email_service_triggered_at DATETIME DEFAULT NULL,
        INDEX idx_invoice_id (invoice_id)
    )`)
	if err != nil {
		return fmt.Errorf("error creating table invoices: %v", err)
	}
	return nil
}

func insertInvoice(invoice *Invoice) error {
	result, err := db.Exec(`INSERT INTO pdf_invoices 
		(product_code, customer_id, invoice_id, email_to, invoice_date, name, address, contact, tax, unit, description, price_per_unit, done_url, price, sub_total, tax_amount, grand_total, currency, currency_symbol) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoice.ProductCode, invoice.CustomerID, invoice.InvoiceID, invoice.EmailTo, invoice.InvoiceDate,
		invoice.Name, invoice.Address, invoice.Contact, invoice.Tax, invoice.Unit, invoice.Description,
		invoice.PricePerUnit, invoice.DoneURL, invoice.Price, invoice.SubTotal, invoice.TaxAmount, invoice.GrandTotal, invoice.Currency, invoice.CurrencySymbol)
	if err != nil {
		return fmt.Errorf("error inserting invoice into database: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error calling LastInsertId: %v", err)
	}
	invoice.ID = int(id)
	return nil
}

// Helper function to retrieve an invoice by invoice ID
func getPdfInvoiceByInvoiceID(invoiceID string) (*Invoice, error) {
	var (
		invoice                 Invoice
		emailServiceTriggeredAt sql.NullString
	)
	err := db.QueryRow("SELECT * FROM pdf_invoices WHERE invoice_id = ?", invoiceID).Scan(
		&invoice.ID, &invoice.ProductCode, &invoice.CustomerID, &invoice.InvoiceID, &invoice.EmailTo, &invoice.InvoiceDate,
		&invoice.Name, &invoice.Address, &invoice.Contact, &invoice.Tax, &invoice.Unit, &invoice.Description,
		&invoice.PricePerUnit, &invoice.Price, &invoice.SubTotal, &invoice.TaxAmount, &invoice.GrandTotal, &invoice.Currency, &invoice.CurrencySymbol, &invoice.DoneURL, &invoice.EmailServiceID, &invoice.EmailServiceMessage,
		&invoice.EmailServiceStatus, &emailServiceTriggeredAt,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, nil
	}

	if emailServiceTriggeredAt.Valid {
		t, err := time.Parse(time.DateTime, emailServiceTriggeredAt.String)
		if err != nil {
			return nil, err
		}
		invoice.EmailServiceTriggeredAt = &t
	}

	return &invoice, nil
}

// Helper function to retrieve an invoice by ID
func getPdfInvoiceByID(ID int) (*Invoice, error) {
	var (
		invoice                 Invoice
		emailServiceTriggeredAt sql.NullString
	)
	err := db.QueryRow("SELECT * FROM pdf_invoices WHERE id = ?", ID).Scan(
		&invoice.ID,
		&invoice.ProductCode, &invoice.CustomerID, &invoice.InvoiceID,
		&invoice.EmailTo, &invoice.InvoiceDate,
		&invoice.Name, &invoice.Address, &invoice.Contact,
		&invoice.Tax, &invoice.Unit, &invoice.Description, &invoice.PricePerUnit,
		&invoice.Price, &invoice.SubTotal, &invoice.TaxAmount,
		&invoice.GrandTotal, &invoice.Currency, &invoice.CurrencySymbol,
		&invoice.DoneURL,
		&invoice.EmailServiceID, &invoice.EmailServiceMessage,
		&invoice.EmailServiceStatus, &emailServiceTriggeredAt,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, nil
	}

	if emailServiceTriggeredAt.Valid {
		t, err := time.Parse(time.DateTime, emailServiceTriggeredAt.String)
		if err != nil {
			return nil, err
		}
		invoice.EmailServiceTriggeredAt = &t
	}

	return &invoice, nil
}

// Helper function to update an existing invoice record
func updateInvoice(invoice Invoice) error {
	_, err := db.Exec(`UPDATE pdf_invoices SET 
		product_code = ?, customer_id = ?, email_to = ?, invoice_date = ?, name = ?, address = ?, contact = ?, 
		tax = ?, unit = ?, description = ?, price_per_unit = ?, price = ?, sub_total = ?, tax_amount = ?, grand_total = ?, currency = ?, currency_symbol = ?, done_url = ?
		WHERE id = ?`,
		invoice.ProductCode, invoice.CustomerID, invoice.EmailTo, invoice.InvoiceDate,
		invoice.Name, invoice.Address, invoice.Contact, invoice.Tax, invoice.Unit, invoice.Description,
		invoice.PricePerUnit, invoice.Price, invoice.SubTotal, invoice.TaxAmount, invoice.GrandTotal, invoice.Currency, invoice.CurrencySymbol, invoice.DoneURL, invoice.ID,
	)
	if err != nil {
		return fmt.Errorf("error updating invoice in database: %v", err)
	}
	return nil
}

func updateEmailServiceFieldsByID(id int, emailServiceID *int, emailServiceMessage string, emailServiceStatus int16, emailServiceTriggeredAt string) error {
	// Construct the SQL query
	query := `UPDATE pdf_invoices SET email_service_id = ?, email_service_message = ?, email_service_status = ?, email_service_triggered_at = ? WHERE id = ?`

	// Execute the SQL query with parameters
	_, err := db.Exec(query, emailServiceID, emailServiceMessage, emailServiceStatus, emailServiceTriggeredAt, id)
	if err != nil {
		return err
	}

	return nil
}
