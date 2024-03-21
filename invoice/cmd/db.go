package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB

// Status represents the status of a subscription.
type Status int8

// Define constants for different subscription statuses.
const (
	StatusNotStarted Status = iota
	StatusProcessing
	StatusDone
	StatusFailed
)

// String returns the string representation of the status.
func (s Status) String() string {
	switch s {
	case StatusNotStarted:
		return "NOT_STARTED"
	case StatusProcessing:
		return "PROCESSING"
	case StatusDone:
		return "DONE"
	case StatusFailed:
		return "FAILED"
	default:
		return fmt.Sprintf("Unknown status: %d", s)
	}
}

// Value implements the driver.Valuer interface.
func (s Status) Value() (driver.Value, error) {
	return int64(s), nil
}

// Scan implements the sql.Scanner interface.
func (s *Status) Scan(value interface{}) error {
	if value == nil {
		*s = StatusNotStarted
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = Status(v)
		return nil
	case []byte:
		*s = Status(v[0])
		return nil
	default:
		return fmt.Errorf("unsupported type for Status: %T", value)
	}
}

type Subscription struct {
	ID                      int        `json:"id"`
	CustomerID              string     `json:"customer_id"`
	ContractStartDate       time.Time  `json:"contract_start_date"`
	Duration                int        `json:"duration"`
	DurationUnits           string     `json:"duration_units"`
	BillingFrequency        int        `json:"billing_frequency"`
	BillingFrequencyUnits   string     `json:"billing_frequency_units"`
	Price                   float64    `json:"price"`
	Tax                     int        `json:"tax"`
	Currency                string     `json:"currency"`
	ProductCode             string     `json:"product_code"`
	BillingFrequencyRemains int        `json:"billing_frequency_remains"`
	NextInvoiceDate         time.Time  `json:"next_invoice_date"`
	InvoicingStartedAt      *time.Time `json:"invoicing_started_at,omitempty"`
	Status                  Status     `json:"status"`
}

// Invoice represents the invoice entity in the database.
type Invoice struct {
	ID                 int       `json:"id"`
	SubscriptionID     int       `json:"subscription_id"`
	CustomerID         string    `json:"customer_id"`
	ProductCode        string    `json:"product_code"`
	EmailTo            string    `json:"emailTo"`
	InvoiceDate        time.Time `json:"invoiceDate"`
	Name               string    `json:"name"`
	Address            string    `json:"address"`
	Contact            string    `json:"contact"`
	Tax                int       `json:"tax"`
	Unit               int       `json:"unit"`
	Description        string    `json:"description"`
	PricePerUnit       float64   `json:"pricePerUnit"`
	Price              float64   `json:"price"`
	SubTotal           float64   `json:"subTotal"`
	TaxAmount          float64   `json:"taxAmount"`
	GrandTotal         float64   `json:"grandTotal"`
	Currency           string    `json:"currency"`
	CurrencySymbol     string    `json:"currencySymbol"`
	InvoicingStartedAt time.Time `json:"invoicing_started_at"`
	Status             Status    `json:"status"`
}

func createTable(db *sql.DB) error {
	// status 0 => NOT_STARTED, 1 => PROCESSING, 2 => DONE, 3 => FAILED
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
			id INT AUTO_INCREMENT PRIMARY KEY,
			customer_id VARCHAR(255) NOT NULL,
			contract_start_date DATE NOT NULL,
			duration INT NOT NULL,
			duration_units VARCHAR(255) NOT NULL,
			billing_frequency INT NOT NULL,
			billing_frequency_units VARCHAR(255) NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			tax INT NOT NULL,
			currency VARCHAR(3) NOT NULL,
			product_code VARCHAR(255) NOT NULL,
			billing_frequency_remains INT NOT NULL,
			next_invoice_date DATE NOT NULL,
			invoicing_started_at DATETIME DEFAULT NULL,
			status TINYINT NOT NULL DEFAULT 0,
			INDEX idx_billing_frequency_remains (billing_frequency_remains),
			INDEX idx_next_invoice_date (next_invoice_date),
			INDEX idx_status (status)
	)`)
	if err != nil {
		return fmt.Errorf("error creating table: %v", err)
	}

	// status 0 => NOT_STARTED, 1 => PROCESSING, 2 => DONE, 3 => FAILED
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS invoices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    subscription_id INT NOT NULL,
    customer_id VARCHAR(255) NOT NULL,
		product_code VARCHAR(255) NOT NULL,
		email_to VARCHAR(255) NOT NULL,
		invoice_date DATE NOT NULL,
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
		invoicing_started_at DATETIME NOT NULL,
		status TINYINT NOT NULL DEFAULT 1,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE ON UPDATE CASCADE,
		INDEX idx_customer_id (customer_id),
		INDEX idx_product_code (product_code),
		INDEX idx_invoicing_started_at (invoicing_started_at),
		INDEX idx_status (status)
	)`)
	if err != nil {
		return fmt.Errorf("error creating table: %v", err)
	}
	return nil
}

// GetSubscriptions retrieves subscriptions from the database based on the specified criteria.
func GetSubscriptions(db *sql.DB, currentTime time.Time) ([]Subscription, error) {
	// Query to retrieve subscriptions
	query := `
		SELECT id, customer_id, contract_start_date, duration, duration_units, 
			billing_frequency, billing_frequency_units, price, tax, currency, 
			product_code, billing_frequency_remains, next_invoice_date, 
			invoicing_started_at, status
		FROM subscriptions
		WHERE billing_frequency_remains > 0 
			AND next_invoice_date <= ? 
			AND (status != ? AND status != ?)
		ORDER BY id ASC
		LIMIT 10
	`

	// Execute the query
	rows, err := db.Query(query, currentTime.Format(time.DateTime), StatusProcessing, StatusFailed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over the results and scan into Subscription structs
	var subscriptions []Subscription
	for rows.Next() {
		var (
			subscription Subscription
			s            sql.NullString
		)
		err := rows.Scan(
			&subscription.ID,
			&subscription.CustomerID,
			&subscription.ContractStartDate,
			&subscription.Duration,
			&subscription.DurationUnits,
			&subscription.BillingFrequency,
			&subscription.BillingFrequencyUnits,
			&subscription.Price,
			&subscription.Tax,
			&subscription.Currency,
			&subscription.ProductCode,
			&subscription.BillingFrequencyRemains,
			&subscription.NextInvoiceDate,
			&s,
			&subscription.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning subscription row: %v", err)
		}
		if s.Valid {
			t, err := time.Parse(time.DateTime, s.String)
			if err != nil {
				return nil, fmt.Errorf("error parsing invoicing_started_at: %v", err)
			}
			subscription.InvoicingStartedAt = &t
		}
		subscriptions = append(subscriptions, subscription)
	}

	// Check for any errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func UpdateSubscriptionStatus(tx *sql.Tx, invoicingStartedAt time.Time, status Status, id int) error {
	query := `
		UPDATE subscriptions 
		SET invoicing_started_at = ?, status = ? 
		WHERE id = ?
	`
	_, err := tx.Exec(query, invoicingStartedAt.Format(time.DateTime), status, id)
	if err != nil {
		return fmt.Errorf("error updating subscription status: %v", err)
	}
	return nil
}

func UpdateSubscriptionFields(tx *sql.Tx, id int, billingRemains int, status Status, nextInvoiceDate time.Time) error {
	query := `
		UPDATE subscriptions 
		SET billing_frequency_remains = ?, 
		    next_invoice_date = ?, 
		    invoicing_started_at = NULL, 
		    status = ? 
		WHERE id = ?
	`
	_, err := tx.Exec(query, billingRemains, nextInvoiceDate.Format(time.DateOnly), status, id)
	if err != nil {
		return fmt.Errorf("error completing subscription: %v", err)
	}
	return nil
}

// InsertInvoice inserts a new invoice into the database.
func InsertInvoice(tx *sql.Tx, invoice *Invoice) error {
	// Prepare the SQL statement for inserting an invoice
	query := `
		INSERT INTO invoices (subscription_id, customer_id, product_code, email_to,
			invoice_date, name, address, contact, tax, unit, description, price_per_unit,
			price, sub_total, tax_amount, grand_total, currency, currency_symbol,
			invoicing_started_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Execute the SQL statement with the provided values
	result, err := tx.Exec(query, invoice.SubscriptionID, invoice.CustomerID, invoice.ProductCode,
		invoice.EmailTo, invoice.InvoiceDate, invoice.Name, invoice.Address, invoice.Contact,
		invoice.Tax, invoice.Unit, invoice.Description, invoice.PricePerUnit, invoice.Price,
		invoice.SubTotal, invoice.TaxAmount, invoice.GrandTotal, invoice.Currency,
		invoice.CurrencySymbol, invoice.InvoicingStartedAt, invoice.Status)
	if err != nil {
		return fmt.Errorf("error inserting invoice: %v", err)
	}

	// Get the ID of the newly inserted invoice
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last inserted ID: %v", err)
	}

	// Set the ID of the invoice
	invoice.ID = int(id)

	return nil
}

// GetInvoiceID returns the ID of the invoice in the specified format.
func (i *Invoice) GetInvoiceID() string {
	return fmt.Sprintf("INV-%d-%s-%s-%d", i.SubscriptionID, i.CustomerID, i.ProductCode, i.ID)
}

// ParseInvoiceID parses an invoice ID and validates the format.
func ParseInvoiceID(invoiceID string) (*Invoice, error) {
	parts := strings.Split(invoiceID, "-")
	if len(parts) != 5 || parts[0] != "INV" {
		return nil, fmt.Errorf("invalid invoice ID format")
	}

	subscriptionID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid subscription ID in invoice ID")
	}

	id, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, fmt.Errorf("invalid ID in invoice ID")
	}

	return &Invoice{
		ID:             id,
		SubscriptionID: subscriptionID,
		CustomerID:     parts[2],
		ProductCode:    parts[3],
	}, nil
}

// GetInvoiceByInfo retrieves an invoice from the database based on its ID, subscription ID, customer ID, and product code.
func GetInvoiceByInfo(db *sql.DB, id int, subscriptionID int, customerID string, productCode string) (*Invoice, error) {
	// Query to retrieve the invoice
	query := `
		SELECT id, subscription_id, customer_id, product_code, email_to, invoice_date, 
		name, address, contact, tax, unit, description, price_per_unit, price, sub_total, 
		tax_amount, grand_total, currency, currency_symbol, invoicing_started_at, status
		FROM invoices
		WHERE id = ? AND subscription_id = ? AND customer_id = ? AND product_code = ? AND status != ?
	`

	// Execute the query
	row := db.QueryRow(query, id, subscriptionID, customerID, productCode, StatusFailed)

	// Scan the row into an Invoice struct
	var invoice Invoice
	err := row.Scan(
		&invoice.ID,
		&invoice.SubscriptionID,
		&invoice.CustomerID,
		&invoice.ProductCode,
		&invoice.EmailTo,
		&invoice.InvoiceDate,
		&invoice.Name,
		&invoice.Address,
		&invoice.Contact,
		&invoice.Tax,
		&invoice.Unit,
		&invoice.Description,
		&invoice.PricePerUnit,
		&invoice.Price,
		&invoice.SubTotal,
		&invoice.TaxAmount,
		&invoice.GrandTotal,
		&invoice.Currency,
		&invoice.CurrencySymbol,
		&invoice.InvoicingStartedAt,
		&invoice.Status,
	)
	if err != nil {
		return nil, err
	}

	return &invoice, nil
}

// GetSubscriptionByIDCustomerIDProductCode retrieves a subscription from the database based on the specified ID, CustomerID, and ProductCode.
func GetSubscriptionByIDCustomerIDProductCode(db *sql.DB, subscriptionID int, customerID, productCode string) (*Subscription, error) {
	// Query to retrieve the subscription
	query := `
			SELECT id, customer_id, contract_start_date, duration, duration_units, 
					billing_frequency, billing_frequency_units, price, tax, currency, 
					product_code, billing_frequency_remains, next_invoice_date, 
					invoicing_started_at, status
			FROM subscriptions
			WHERE id = ? AND customer_id = ? AND product_code = ? AND status != ?
	`

	// Execute the query
	row := db.QueryRow(query, subscriptionID, customerID, productCode, StatusFailed)

	// Scan the result into a Subscription struct
	var (
		subscription Subscription
		t            sql.NullTime
	)
	err := row.Scan(
		&subscription.ID,
		&subscription.CustomerID,
		&subscription.ContractStartDate,
		&subscription.Duration,
		&subscription.DurationUnits,
		&subscription.BillingFrequency,
		&subscription.BillingFrequencyUnits,
		&subscription.Price,
		&subscription.Tax,
		&subscription.Currency,
		&subscription.ProductCode,
		&subscription.BillingFrequencyRemains,
		&subscription.NextInvoiceDate,
		&t,
		&subscription.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, fmt.Errorf("error retrieving subscription: %v", err)
	}

	if t.Valid {
		subscription.InvoicingStartedAt = &t.Time
	}

	return &subscription, nil
}

func SetStatusInvoice(tx *sql.Tx, id int, status Status) error {
	query := `
		UPDATE invoices 
		SET status = ? 
		WHERE id = ?
	`
	_, err := tx.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("error setting status for invoice: %v", err)
	}
	return nil
}

func GetInvoices(db *sql.DB, InvoicingStartedAt time.Time) ([]Invoice, error) {
	query := `
			SELECT id, subscription_id, customer_id, product_code, email_to, invoice_date, 
						 name, address, contact, tax, unit, description, price_per_unit, price, 
						 sub_total, tax_amount, grand_total, currency, currency_symbol, 
						 invoicing_started_at, status
			FROM invoices
			WHERE invoicing_started_at <= ? AND status = ?
			LIMIT 100
	`

	rows, err := db.Query(query, InvoicingStartedAt.Format(time.DateTime), StatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve invoices: %w", err)
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var invoice Invoice
		if err := rows.Scan(
			&invoice.ID,
			&invoice.SubscriptionID,
			&invoice.CustomerID,
			&invoice.ProductCode,
			&invoice.EmailTo,
			&invoice.InvoiceDate,
			&invoice.Name,
			&invoice.Address,
			&invoice.Contact,
			&invoice.Tax,
			&invoice.Unit,
			&invoice.Description,
			&invoice.PricePerUnit,
			&invoice.Price,
			&invoice.SubTotal,
			&invoice.TaxAmount,
			&invoice.GrandTotal,
			&invoice.Currency,
			&invoice.CurrencySymbol,
			&invoice.InvoicingStartedAt,
			&invoice.Status,
		); err != nil {
			return nil, fmt.Errorf("error scanning invoice row: %w", err)
		}
		invoices = append(invoices, invoice)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over invoice rows: %w", err)
	}

	return invoices, nil
}
