### Customer API Service

#### Overview
The API allows users to retrieve customer information based on a customer ID.

#### Features
- Retrieve customer information by customer ID.
- Serve customer data in JSON format.
- Error handling for non-existent customer IDs.

#### Details

##### Dependencies
- [Chi](https://github.com/go-chi/chi/v5)

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

##### Environment Variables
The following environment variables are required to run the project:
- `PORT`: Specifies the port on which the server will listen.

#### Running the Application
1. Set the required environment variables:
   ```
   export PORT=8080
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Build and run the application:
   ```
   go run main.go
   ```

#### Example Usage
1. Retrieve customer information by customer ID:
   ```
   GET /api/customers/CUSTOMER-0001
   ```

#### Summary
The Customer API provides an in memory dummy simple service to retrieve customer information using a RESTful interface. It uses the Chi router for routing and handling HTTP requests.
