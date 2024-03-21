### Technical Document: Customer API

#### Overview
This document provides an overview and analysis of the Customer API implemented in Go using the Chi router. The API allows users to retrieve customer information based on a customer ID.

#### Features
- Retrieve customer information by customer ID.
- Serve customer data in JSON format.
- Error handling for non-existent customer IDs.

#### Implementation Details

##### Dependencies
- [Chi](https://github.com/go-chi/chi/v5): A lightweight, idiomatic and composable router for building Go HTTP services.

##### Customer Struct
The `Customer` struct represents customer data with the following fields:
- `Name`: Name of the customer.
- `Email`: Email address of the customer.
- `Address`: Address of the customer.
- `Contact`: Contact number of the customer.

##### Data Store
Customer data is stored in a in memory map called `customerData`, where each key represents a customer ID and its corresponding value is a `Customer` struct containing the customer's information.

##### API Routes
1. **GET /api/customers/{customerID}**: Retrieves customer information based on the provided customer ID.
   - If the customer ID exists in the `customerData` map, the corresponding customer information is returned in JSON format.
   - If the customer ID does not exist, a 404 Not Found error is returned.

##### Running the Server
The server listens on the port specified by the `PORT` environment variable. Ensure the `PORT` environment variable is set before running the server.

#### Running the Application
1. Install dependencies:
   ```
   go mod tidy
   ```

2. Build and run the application:
   ```
   go run main.go
   ```

#### Example Usage
1. Retrieve customer information by customer ID:
   ```
   GET /api/customers/CUSTOMER-0001
   ```

#### Conclusion
The Customer API provides a simple and efficient way to retrieve customer information using a RESTful interface. It leverages the Chi router for routing and handling HTTP requests, making it easy to extend and maintain.