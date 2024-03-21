### Invoice Service and CRON

#### Overview
This Go service runs a CRON server and a REST api to handle invoicing. It handles creation of invoice in the database and calls PDF service to generate PDF.

#### Features
- Runs CRON to handle invoicing and cleans up stalled invoices
- Store information in a MySQL database.
- Handle callback requests from the pdf service
- Graceful shutdown of the server.

#### Details

##### Files

1. **cron.go:**
   - This file contains functions for processing invoices on a scheduled basis.
   - `processStalledInvoices`: Marks invoices that have been processing for longer than 10 minutes as failed.
   - `processInvoiceDaily`: Generates invoices for pending subscriptions and updates their statuses.

2. **db.go:**
   - Handles database operations including table creation, data insertion, retrieval, and updates related to subscriptions and invoices.
   - Defines database schema and table structures for subscriptions and invoices.
   - Implements functions for querying subscriptions and invoices based on various criteria.
   - Provides methods for updating subscription and invoice statuses.

3. **helpers.go:**
   - Contains helper functions for interacting with external services and parsing HTTP responses.
   - Defines structs for account and customer details retrieved from external services.
   - Implements functions for fetching account and customer details using HTTP requests.
   - Provides utility functions for sending HTTP requests with various methods (GET, POST) and handling responses.

4. **main.go:**
  - This file serves as the entry point for the application.
  - It initializes the database connection, sets up any required configurations, and starts the application.


##### Database Schema
The project uses a relational database with two main tables: `subscriptions` and `invoices`.

**Subscriptions Table:**

- `id`: INT (Primary Key)
- `customer_id`: VARCHAR(255)
- `contract_start_date`: DATE
- `duration`: INT
- `duration_units`: VARCHAR(255)
- `billing_frequency`: INT
- `billing_frequency_units`: VARCHAR(255)
- `price`: DECIMAL(10, 2)
- `tax`: INT
- `currency`: VARCHAR(3)
- `product_code`: VARCHAR(255)
- `billing_frequency_remains`: INT
- `next_invoice_date`: DATE
- `invoicing_started_at`: DATETIME
- `status`: TINYINT

**Invoices Table:**

- `id`: INT (Primary Key)
- `subscription_id`: INT (Foreign Key)
- `customer_id`: VARCHAR(255)
- `product_code`: VARCHAR(255)
- `email_to`: VARCHAR(255)
- `invoice_date`: DATE
- `name`: VARCHAR(255)
- `address`: VARCHAR(255)
- `contact`: VARCHAR(255)
- `tax`: INT
- `unit`: INT
- `description`: VARCHAR(255)
- `price_per_unit`: DECIMAL(10, 2)
- `price`: DECIMAL(10, 2)
- `sub_total`: DECIMAL(10, 2)
- `tax_amount`: DECIMAL(10, 2)
- `grand_total`: DECIMAL(10, 2)
- `currency`: VARCHAR(3)
- `currency_symbol`: VARCHAR(5)
- `invoicing_started_at`: DATETIME
- `status`: TINYINT

##### Callback Architecture

The project follows a callback architecture for processing subscriptions and generating invoices.

1. **Process Stalled Invoices**: A cron job runs hourly to mark invoices that have taken longer than 10 minutes to process as failed. This is handled by the `processStalledInvoices` function.

2. **Process Invoice Daily**: Another cron job runs daily to process pending subscriptions and generate invoices. This is handled by the `processInvoiceDaily` function.

3. **Callback URLs**: After generating invoices, the application calls a PDF service to generate PDF invoices. Upon completion, a callback URL is invoked with the status of the invoice generation process.

##### Handling Failure and Success

- **Failure Handling**:
  - In case of errors during database operations or external service calls, appropriate error messages are logged, and transactions are rolled back to maintain data consistency.
  - Failed invoices are marked with a status of `StatusFailed` in the database, and subscriptions are updated accordingly.

- **Success Handling**:
  - Successful invoice generation updates the status of the invoice to `StatusDone`.
  - Subscriptions are updated with the next invoice date and remaining billing frequency if the invoice generation is successful.

##### Environment Variables

To run the project, the following environment variables need to be configured:

- **MYSQL_DSN**: The URL of the database.
- **ACCOUNTS_SERVICE_BASE_URL**: The base URL of the accounts service.
- **CUSTOMER_SERVICE_BASE_URL**: The base URL of the customer service.
- **PDF_SVC**: The URL of the PDF generation service.
- **BASE_URL**: The base URL of the application.
- **PORT**: Port number on which the server will listen.

##### Callback Architecture

The project follows a callback architecture for processing subscriptions and generating invoices.

1. **Process Stalled Invoices**: A cron job runs hourly to mark invoices that have taken longer than 10 minutes to process as failed. This is handled by the `processStalledInvoices` function.

2. **Process Invoice Daily**: Another cron job runs daily to process pending subscriptions and generate invoices. This is handled by the `processInvoiceDaily` function.

3. **Callback URLs**: After generating invoices, the application calls a PDF service to generate PDF invoices. Upon completion, a callback URL is invoked with the status of the invoice generation process.

##### Handling Failure and Success

- **Failure Handling**:
  - In case of errors during database operations or external service calls, appropriate error messages are logged, and transactions are rolled back to maintain data consistency.
  - Failed invoices are marked with a status of `StatusFailed` in the database, and subscriptions are updated accordingly.

- **Success Handling**:
  - Successful invoice generation updates the status of the invoice to `StatusDone`.
  - Subscriptions are updated with the next invoice date and remaining billing frequency if the invoice generation is successful.

#### Running the Application
1. Set the required environment variables

2. Ensure that the MySQL database is set up and accessible. The service will create the necessary tables automatically if it doesn't exist.

3. Build and run the application:

   ```bash
   go run ./invoice/cmd/*.go
   ```

#### Summary
The invoice service runs CRON and a callback REST api to generate invoice process and make necessary changes in database.