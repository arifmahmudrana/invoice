package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/arifmahmudrana/invoice/email"
	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB
var mutex sync.Mutex

type Email struct {
	ID            int          `json:"id"`
	ProductCode   string       `json:"productCode"`
	CustomerID    string       `json:"customerID"`
	InvoiceID     string       `json:"invoiceID"`
	EmailTo       string       `json:"emailTo"`
	FileHash      string       `json:"fileHash"`
	DoneURL       string       `json:"doneURL"`
	InvoiceSentAt sql.NullTime `json:"invoiceSentAt"`
	FailedAt      sql.NullTime `json:"failedAt"` // New column
}

// main function
// to run the project use something like
// MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/dbname' PORT=8080 PDF_PATH=full-path/ SMTP_PORT=2525 SMTP_HOST='sandbox.smtp.mailtrap.io' SMTP_USER_NAME=user_name SMTP_PASSWORD=password FROM_EMAIL='amrana83@gmail.com' FROM_NAME='Arif Mahmud Rana' EMAIL_SUBJECT='Invoice for the next the next billing' EMAIL_TEMPLATE_PATH=/home/rana/Desktop/invoice/email/templates go run email/cmd/*.go
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Initialize MySQL database connection
	var err error
	db, err = sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Test the database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	// Create table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS emails (
        id int NOT NULL AUTO_INCREMENT,
        productCode varchar(255) NOT NULL,
        customerID varchar(255) NOT NULL,
        invoiceID varchar(255) NOT NULL,
        emailTo varchar(255) NOT NULL,
        fileHash varchar(255) NOT NULL,
        doneURL varchar(255) NOT NULL,
        invoiceSentAt datetime DEFAULT NULL,
        failedAt datetime DEFAULT NULL,
        PRIMARY KEY (id),
        INDEX invoiceID (invoiceID)
      ) ENGINE=InnoDB DEFAULT CHARSET=utf8`)
	if err != nil {
		log.Fatalf("Error creating table emails: %v", err)
	}

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	r.Post("/api/email-invoice", emailInvoiceHandler)

	// Add route /api/email-invoice/{id} using GET method
	r.Get("/api/email-invoice/{id}", getEmailInvoiceHandler)

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"), // 8080
		Handler: r,
	}

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Start the server in a goroutine
	go func() {
		log.Println("Server listening on port " + os.Getenv("PORT"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error: %v", err)
		}
	}()

	// Block until a signal is received
	<-sigChan

	log.Println("Received termination signal. Shutting down server...")

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	// Close the database connection
	if err := db.Close(); err != nil {
		log.Fatalf("Error closing database connection: %v", err)
	}

	log.Println("Server gracefully stopped")
}

func emailInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // Max file size 10MB
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get form values
	productCode := r.FormValue("productCode")
	customerID := r.FormValue("customerID")
	invoiceID := r.FormValue("invoiceID")
	emailTo := r.FormValue("emailTo")
	fileHash := r.FormValue("fileHash")
	doneURL := r.FormValue("doneURL")

	// Get the file from the request
	file, _, err := r.FormFile("invoiceFile")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if a record with the same invoiceID exists
	var existingFileHash string
	var id int
	dbErr := db.QueryRow("SELECT id, fileHash FROM emails WHERE invoiceID = ?", invoiceID).Scan(&id, &existingFileHash)
	if dbErr != nil && dbErr != sql.ErrNoRows {
		log.Printf("Error checking existing record: %v\n", dbErr)
		http.Error(w, "Error checking existing record", http.StatusInternalServerError)
		return
	}

	// If record with the same invoiceID doesn't exist or fileHash doesn't match, delete previous file and create a new one
	var emailInvoice bool
	assetPath := os.Getenv("PDF_PATH")
	if dbErr == sql.ErrNoRows || existingFileHash != fileHash {
		if existingFileHash != "" {
			if err := os.RemoveAll(assetPath + existingFileHash); err != nil {
				log.Printf("Error deleting previous file: %v\n", err)
				http.Error(w, "Error deleting previous file", http.StatusInternalServerError)
				return
			}
		}

		// Create directory with fileHash name
		dirPath := assetPath + fileHash
		log.Printf("Directory path for the PDF: %s\n", dirPath)
		if err := os.Mkdir(dirPath, 0755); err != nil {
			log.Printf("Error creating directory: %v\n", err)
			http.Error(w, "Error creating directory", http.StatusInternalServerError)
			return
		}

		// Create the file in the directory and move the uploaded file there
		filePath := dirPath + "/invoice.pdf"
		out, err := os.Create(filePath)
		if err != nil {
			log.Printf("Error creating file: %v\n", err)
			http.Error(w, "Error creating file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		// Write the file to disk
		_, err = io.Copy(out, file)
		if err != nil {
			log.Printf("Error writing file to disk: %v\n", err)
			http.Error(w, "Error writing file to disk", http.StatusInternalServerError)
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		var result sql.Result
		var created bool
		if dbErr == sql.ErrNoRows {
			// Insert a new record into the database
			result, err = db.Exec("INSERT INTO emails (productCode, customerID, invoiceID, emailTo, fileHash, doneURL) VALUES (?, ?, ?, ?, ?, ?)", productCode, customerID, invoiceID, emailTo, fileHash, doneURL)
			created = true
		} else {
			// Update existing record in the database with fileHash and set invoiceSentAt to null
			_, err = db.Exec("UPDATE emails SET fileHash = ?, invoiceSentAt = NULL WHERE invoiceID = ?", fileHash, invoiceID)
		}
		if err != nil {
			log.Printf("Error while database operation: %v\n", err)
			http.Error(w, "Error while database operation", http.StatusInternalServerError)
			return
		}
		emailInvoice = true
		if created {
			idRes, err := result.LastInsertId()
			if err != nil {
				log.Printf("Error calling LastInsertId: %v\n", err)
				http.Error(w, "Error calling LastInsertId", http.StatusInternalServerError)
				return
			}
			log.Printf("ID of the new record: %d\n", idRes)
			id = int(idRes)
		}
	}

	// Example: Print the received data
	log.Printf("Received invoice email request:\nProduct Code: %s\nCustomer ID: %s\nInvoice ID: %s\nEmail To: %s\nFile Hash: %s\nDone URL: %s\n",
		productCode, customerID, invoiceID, emailTo, fileHash, doneURL)

	// Example response
	resp := map[string]string{"message": "Invoice email request received and processing"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// Call the retrieveRecord function as a goroutine
	go func(id int, emailInvoice bool) {
		mutex.Lock()
		defer mutex.Unlock()

		if emailInvoice {
			// Retrieve database record for invoiceID with invoiceSentAt null
			em, err := retrieveRecord(id)
			if err != nil {
				return
			}
			if em == nil {
				log.Printf("No record found for id %d\n", id)
				return
			}

			// sent email invoice
			if err := sendEmail(*em); err != nil {
				failedAt := time.Now().UTC().Format(time.DateTime)
				// update database set failedAt
				if _, err := db.Exec("UPDATE emails SET failedAt = ?, invoiceSentAt = NULL WHERE id = ?", failedAt, id); err != nil {
					log.Printf("Error while `UPDATE emails SET failedAt = %s, invoiceSentAt = NULL WHERE id = %d`: %+v\n", failedAt, id, err)
				}

				// call the doneURL with a failedMessage, status
				x := map[string]interface{}{
					"failedMessage": "Failed to process the request",
					"status":        http.StatusInternalServerError,
					"failedAt":      failedAt,
				}
				if err := callDoneURL(em.DoneURL, x); err != nil {
					log.Printf("Error while calling doneURL %s with parameter %#v: %+v\n", em.DoneURL, x, err)
				}
				return
			}

			// update database set failedAt null and invoiceSentAt now
			invoiceSentAt := time.Now().UTC().Format(time.DateTime)
			if _, err := db.Exec("UPDATE emails SET invoiceSentAt = ?, failedAt = NULL WHERE id = ?", invoiceSentAt, id); err != nil {
				log.Printf("Error while `UPDATE emails SET invoiceSentAt = %s, failedAt = NULL WHERE id = %d`: %+v\n", invoiceSentAt, id, err)
			}

			// call the doneURL with a successMessage, status, invoiceSentAt and id of emails table record
			x := map[string]interface{}{
				"successMessage": "Successfully processed the request",
				"status":         http.StatusOK,
				"invoiceSentAt":  invoiceSentAt,
				"ID":             id,
			}
			if err := callDoneURL(em.DoneURL, x); err != nil {
				log.Printf("Error while calling doneURL %s with parameter %#v: %+v\n", em.DoneURL, x, err)
			}
		}
	}(id, emailInvoice)
}

func retrieveRecord(id int) (*Email, error) {
	// Retrieve database record for invoiceID with invoiceSentAt null
	var em Email
	err := db.QueryRow("SELECT * FROM emails WHERE id = ?", id).Scan(&em.ID, &em.ProductCode, &em.CustomerID, &em.InvoiceID, &em.EmailTo, &em.FileHash, &em.DoneURL, &em.InvoiceSentAt, &em.FailedAt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error retrieving record for id %d: %v\n", id, err)
			return &em, err
		}
		return nil, nil
	}

	log.Printf("Retrieved database record for id %d: %+v\n", id, em)

	return &em, nil
}

func sendEmail(em Email) error {
	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		log.Printf("Error while converting SMTP_PORT environment variable to int: %+v\n", err)
		return err
	}
	mail := email.Mail{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     smtpPort,
		Username: os.Getenv("SMTP_USER_NAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}
	log.Printf("mail struct constructed: %v\n", mail)

	x := email.Message{
		From:     os.Getenv("FROM_EMAIL"),
		FromName: os.Getenv("FROM_NAME"),
		To:       em.EmailTo,
		Subject:  os.Getenv("EMAIL_SUBJECT"),
		Attachments: []string{
			os.Getenv("PDF_PATH") + em.FileHash + "/invoice.pdf",
		},
		Data:    template.HTML("<p>Thank you for using our services.</p>"), // HTML for invoice email
		DataMap: nil,
	}
	log.Printf("email.Message struct constructed: %v\n", x)
	err = mail.SendSMTPMessage(x, os.Getenv("EMAIL_TEMPLATE_PATH"))
	if err != nil {
		log.Printf("Error while sending email using SendSMTPMessage: %+v\n", err)

		return err
	}

	return nil
}

// callDoneURL makes a POST request to the specified URL with the given payload.
func callDoneURL(url string, payload map[string]interface{}) error {
	// Marshal payload data to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling payload data: %v\n", err)
		return fmt.Errorf("error marshaling payload data: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("error creating HTTP request: %v\n", err)
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client
	client := &http.Client{}

	// Send HTTP request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending HTTP request: %v\n", err)
		return fmt.Errorf("error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code: %d\n", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func getEmailInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the id parameter from the URL path
	idStr := chi.URLParam(r, "id")

	// Convert the id parameter to an integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Retrieve the database record for the provided id
	em, err := retrieveRecord(id)
	if err != nil {
		log.Printf("Error retrieving record: %v\n", err)
		http.Error(w, "Error retrieving record", http.StatusInternalServerError)
		return
	}

	if em == nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	// sent email invoice
	if err := sendEmail(*em); err != nil {
		log.Printf("Error calling sendEmail: %v\n", err)
		failedAt := time.Now().UTC().Format(time.DateTime)
		// update database set failedAt
		if _, err := db.Exec("UPDATE emails SET failedAt = ?, invoiceSentAt = NULL WHERE id = ?", failedAt, id); err != nil {
			log.Printf("Error while `UPDATE emails SET failedAt = %s, invoiceSentAt = NULL WHERE id = %d`: %+v\n", failedAt, id, err)
		}

		// call the doneURL with a failedMessage, status
		x := map[string]interface{}{
			"failedMessage": "Failed to process the request",
			"status":        http.StatusInternalServerError,
			"failedAt":      failedAt,
		}
		if err := callDoneURL(em.DoneURL, x); err != nil {
			log.Printf("Error while calling doneURL %s with parameter %#v: %+v\n", em.DoneURL, x, err)
		}
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// update database set failedAt null and invoiceSentAt now
	invoiceSentAt := time.Now().UTC().Format(time.DateTime)
	if _, err := db.Exec("UPDATE emails SET invoiceSentAt = ?, failedAt = NULL WHERE id = ?", invoiceSentAt, id); err != nil {
		log.Printf("Error while `UPDATE emails SET invoiceSentAt = %s, failedAt = NULL WHERE id = %d`: %+v\n", invoiceSentAt, id, err)

		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// call the doneURL with a successMessage, status, invoiceSentAt and id of emails table record
	x := map[string]interface{}{
		"successMessage": "Successfully processed the request",
		"status":         http.StatusOK,
		"invoiceSentAt":  invoiceSentAt,
		"ID":             id,
	}
	if err := callDoneURL(em.DoneURL, x); err != nil {
		log.Printf("Error while calling doneURL %s with parameter %#v: %+v\n", em.DoneURL, x, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// Retrieve the database record for the provided id
	em, err = retrieveRecord(id)
	if err != nil {
		log.Printf("Error retrieving record: %v\n", err)
		http.Error(w, "Error retrieving record", http.StatusInternalServerError)
		return
	}

	if em == nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	// Encode the retrieved record as JSON and write it to the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(em)
}
