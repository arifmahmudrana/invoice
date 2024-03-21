### Email Invoice Service

#### Overview
The service handles requests to send invoice emails to customers and on successful email sent or failure.

#### Features
- Send invoice emails with attached PDF files to customers.
- Store email invoice information in a MySQL database.
- Retrieve email invoice information by ID.
- Support for asynchronous processing of email sending and database operations.
- Graceful shutdown of the server.

#### Details

##### Database Structure
The service utilizes a MySQL database to store information related to email invoices. The `emails` table has the following structure:
- `id`: Unique identifier for each email invoice request.
- `productCode`: Code identifying the product associated with the invoice.
- `customerID`: Identifier for the customer receiving the invoice.
- `invoiceID`: Unique identifier for the invoice.
- `emailTo`: Email address of the recipient.
- `fileHash`: Hash of the file associated with the invoice.
- `doneURL`: URL to which callbacks will be made upon completion.
- `invoiceSentAt`: Timestamp indicating when the invoice email was sent.
- `failedAt`: Timestamp indicating when the processing of the email invoice failed.

##### Routes
1. **GET /**: Displays a simple "Hello, World!" message to indicate that the server is running.
2. **POST /api/email-invoice**: Handles requests to send invoice emails. It accepts form data containing details of the invoice and the attached PDF file.
3. **GET /api/email-invoice/{id}**: Retrieves email invoice information by ID and sends invoice email based on the record.

##### Environment Variables
The following environment variables are required to run the project:

- `MYSQL_DSN`: MySQL database connection string.
- `PORT`: Port on which the server will listen.
- `PDF_PATH`: Full path to the directory where PDF files will be stored.
- `SMTP_PORT`: SMTP port for sending emails.
- `SMTP_HOST`: SMTP host for sending emails.
- `SMTP_USER_NAME`: Username for SMTP authentication.
- `SMTP_PASSWORD`: Password for SMTP authentication.
- `FROM_EMAIL`: Email address from which the invoice emails will be sent.
- `FROM_NAME`: Name associated with the sender's email address.
- `EMAIL_SUBJECT`: Subject of the email containing the invoice.
- `EMAIL_TEMPLATE_PATH`: Path to the email template file.

##### Callback Architecture
Upon successful or failed processing of an email invoice request, the service performs a callback to the specified `doneURL`. Callbacks include relevant information such as success or failure messages, status codes, timestamps, and the ID of the corresponding database record.

##### Handling Failure
- If an error occurs during processing, the service updates the database record with a timestamp indicating the failure (`failedAt`), and sets the `invoiceSentAt` field to null.
- A callback is made to the specified `doneURL` with a failure message, status code, and timestamp.
- If an error occurs while sending the email, the service logs the error and updates the database accordingly.

#### Running the Application
1. Set the required environment variables:

   ```bash
   export MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/dbname'
   export PORT=8080
   export PDF_PATH='/full/path/to/pdf/directory/'
   export SMTP_PORT=2525
   export SMTP_HOST='smtp.mailtrap.io'
   export SMTP_USER_NAME='user_name'
   export SMTP_PASSWORD='password'
   export FROM_EMAIL='amrana83@gmail.com'
   export FROM_NAME='Arif Mahmud Rana'
   export EMAIL_SUBJECT='Invoice for the next billing'
   export EMAIL_TEMPLATE_PATH='/path/to/email/template'
   ```

2. Build and run the application:

   ```bash
   go run main.go
   ```

#### Summary
The Email Invoice Service sends invoice emails to customers. It uses Go's concurrency features to handle email sending and database operations asynchronously.