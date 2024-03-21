package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
)

var mutex sync.Mutex

const cbURLPath = "/api/cb-invoice-pdf"

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
	if err := createTable(); err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	r := chi.NewRouter()

	r.Post("/api/generate-invoice-pdf", GenerateInvoicePDFHandler)
	r.Get("/api/invoice-pdf/{id}", InvoicePDFByIDHandler)
	r.Post(cbURLPath+"/{id}", CBInvoicePdfHandler)

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"), // 8080
		Handler: r,
	}

	// Start the server in a separate goroutine
	go func() {
		log.Println("Server listening on port " + os.Getenv("PORT"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error: %v", err)
		}
	}()

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Received termination signal. Shutting down server...")

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
