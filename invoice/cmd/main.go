package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
)

const cbURLPath = "/api/cb"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Initialize MySQL database connection
	var err error
	db, err = sql.Open("mysql", os.Getenv("MYSQL_DSN")) // "user:password@tcp(127.0.0.1:3306)/dbname"
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Test the database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	// Create table if not exists
	if err := createTable(db); err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// Create cron scheduler
	c := cron.New()

	// Add scheduled tasks to the cron scheduler
	c.AddFunc("@hourly", processStalledInvoices)
	c.AddFunc("@daily", processInvoiceDaily)

	// Start cron scheduler in a separate goroutine
	go c.Start()
	defer c.Stop()

	// Create HTTP server with Chi router
	r := chi.NewRouter()

	// Define REST API routes
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Post(cbURLPath+"/{invoiceID}", func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var requestBody struct {
			ID                      int    `json:"id"`
			EmailServiceID          *int   `json:"emailServiceID,omitempty"`
			EmailServiceMessage     string `json:"emailServiceMessage"`
			EmailServiceStatus      int16  `json:"emailServiceStatus"`
			EmailServiceTriggeredAt string `json:"emailServiceTriggeredAt"`
		}
		err = json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			log.Printf("Failed to parse request body: %v\n", err)
			http.Error(w, "Failed to parse request body", http.StatusBadRequest)
			return
		}

		invoiceID := chi.URLParam(r, "invoiceID")
		invoice, err := ParseInvoiceID(invoiceID)
		if err != nil {
			log.Printf("Error ParseInvoiceID for invoiceID %s: %v\n", invoiceID, err)
			http.NotFound(w, r)
			return
		}
		log.Printf("Invoice after parsing: %#v\n", invoice)

		invoice, err = GetInvoiceByInfo(db, invoice.ID, invoice.SubscriptionID, invoice.CustomerID, invoice.ProductCode)
		if err != nil {
			log.Printf("Error GetInvoiceByInfo: %v\n", err)
			http.NotFound(w, r)
			return
		}

		subscription, err := GetSubscriptionByIDCustomerIDProductCode(db, invoice.SubscriptionID, invoice.CustomerID, invoice.ProductCode)
		if err != nil {
			log.Printf("Error GetSubscriptionByIDCustomerIDProductCode: %v\n", err)
			http.NotFound(w, r)
			return
		}

		// Begin the transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error calling Begin for transaction: %v\n", err)
			http.Error(w, "Error calling Begin for transaction", http.StatusInternalServerError)
			return
		}
		nextInvoiceDate := subscription.NextInvoiceDate
		status := StatusFailed
		billingFrequencyRemains := subscription.BillingFrequencyRemains
		if requestBody.EmailServiceStatus == http.StatusOK {
			nextInvoiceDate, err = getNextInvoiceDate(*subscription)
			if err != nil {
				log.Printf("Error calling getNextInvoiceDate: %v\n", err)
				if err := tx.Rollback(); err != nil {
					log.Printf("Error calling transaction Rollback: %v\n", err)
				}
				http.Error(w, "Error calling getNextInvoiceDate", http.StatusInternalServerError)
				return
			}
			status = StatusDone
			billingFrequencyRemains = subscription.BillingFrequencyRemains - 1
		}
		if err = SetStatusInvoice(tx, invoice.ID, status); err != nil {
			log.Printf("Error calling SetStatusInvoice: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			http.Error(w, "Error calling SetStatusInvoice", http.StatusInternalServerError)
			return
		}

		if err = UpdateSubscriptionFields(tx, subscription.ID, billingFrequencyRemains, status, nextInvoiceDate); err != nil {
			log.Printf("Error calling UpdateSubscriptionFields: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			http.Error(w, "Error calling UpdateSubscriptionFields", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			// Rollback the transaction if commit fails and log the error
			log.Printf("Error calling transaction Commit: %v\n", err)
			if err := tx.Rollback(); err != nil {
				log.Printf("Error calling transaction Rollback: %v\n", err)
			}
			http.Error(w, "Error calling transaction Commit", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
	})

	// Start the HTTP server
	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"), // Use PORT environment variable
		Handler: r,
	}

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a separate goroutine
	go func() {
		log.Printf("Server listening on port %s\n", os.Getenv("PORT"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Block until a signal is received
	<-sigChan

	log.Println("Received termination signal. Shutting down server...")

	// Create a context with a timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the HTTP server gracefully
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}

	log.Println("Server gracefully stopped")
}
