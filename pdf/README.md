### PDF generate Service

#### Overview
This Go service is designed for generating PDF invoices and handling callback requests from an email service. The service utilizes a MySQL database to store invoice data and provides RESTful APIs for invoice PDF generation and regenerate by the ID of the record.

#### Features
- Generates PDF file for invoice and sends it to email service.
- Store information in a MySQL database so that it can regenerate and resend the PDF to email service.
- Support for asynchronous processing of PDF generation and calling of email service and database operations.
- Handle callback requests from the email service
- Graceful shutdown of the server.

#### Details

##### Files

- **main.go**: Entry point of the application. Sets up HTTP server, handles termination signals, and manages server shutdown.
- **handlers.go**: Contains HTTP request handlers for generating PDF invoices and handling callback requests from the email service.
- **db.go**: Provides functions for interacting with the MySQL database, including table creation, insertion, and retrieval of invoice data.
- **helpers.go**: Contains helper functions for generating PDF invoices, calculating SHA-1 hash, and sending API requests to the email service.

##### Database Schema
The service uses a MySQL database with the following table schema:

```sql
CREATE TABLE IF NOT EXISTS pdf_invoices (
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
)
```
##### Endpoints

###### 1. Generate Invoice PDF

- **URL**: `POST /api/generate-invoice-pdf`
- **Description**: Generates a PDF invoice based on the provided invoice data.
- **Request Body**: JSON object containing invoice details.
- **Example Request**:
  ```json
  {
    "productCode": "PROD001",
    "customerID": "CUST001",
    "invoiceID": "INV001",
    "emailTo": "customer@example.com",
    "invoiceDate": "2024-03-18",
    "name": "John Doe",
    "address": "123 Main St, City",
    "contact": "+1234567890",
    "tax": 10,
    "unit": 2,
    "description": "Product Description",
    "pricePerUnit": 50.00,
    "currency": "USD",
    "currencySymbol": "$",
    "doneURL": "http://example.com/callback"
  }
  ```
- **Response**: HTTP status code indicating success or failure.

###### 2. Regenerate Invoice PDF by ID

- **URL**: `GET /api/invoice-pdf/{id}`
- **Description**: Regenerates a PDF for invoice by retrieving record from `pdf_invoices` table by ID and sends to email service.
- **Parameters**:
  - **id**: ID of the `pdf_invoices` to retrieve.
- **Response**: HTTP status code indicating success or failure.

###### 3. Callback Invoice PDF Generation Status

- **URL**: `POST /api/cb-invoice-pdf/{id}`
- **Description**: Handles callback requests from the email service regarding the status of invoice PDF generation.
- **Parameters**:
  - **id**: ID of the `pdf_invoices` for which the callback is received.
- **Request Body**: JSON object containing the status and other relevant information.
- **Example Request**:
  ```json
  {
    "status": 200,
    "successMessage": "Invoice PDF generated successfully",
    "invoiceSentAt": "2024-03-18 12:00:00",
    "ID": 123
  }
  ```
- **Response**: HTTP status code indicating success or failure.

##### Environment Variables
The following environment variables are required to run the project:

- **MYSQL_DSN**: MySQL connection string in the format `"user:password@tcp(host:port)/dbname"`.
- **PORT**: Port number on which the server will listen.
- **EMAIL_SVC**: URL of the email service endpoint for sending invoices.
- **BASE_URL**: Base URL of the service.
- **COMPANY_NO**: Company registration number.
- **COMPANY_NAME**: Name of the company.
- **COMPANY_ADDRESS**: Address of the company.
- **COMPANY_CONTACT**: Contact information of the company.
- **COMPANY_LOGO_PATH**: Path to the company logo file.
- **COMPANY_LOGO_IMG_TYPE**: Type of the company logo image (e.g., "png", "jpg").

##### Callback Architecture
Upon successful or failed database record is updated with email service information and propagated to the service it was called by using `doneURL`. Callbacks include relevant information such as success or failure messages, status codes, timestamps, and the ID of the corresponding database record.

#### Running the Application
1. Set the required environment variables

2. Ensure that the MySQL database is set up and accessible. The service will create the necessary table (`pdf_invoices`) automatically if it doesn't exist.

3. Build and run the application:

   ```bash
   go run ./pdf/cmd/*.go
   ```

#### Summary
The PDF Generation Service provides a way to generate and send PDF to email service from a JSON payload. It has endpoint to regenerate PDF from database record and handle callback and notify the callee service.