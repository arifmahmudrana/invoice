### Account API Service

#### Overview
The API allows users to retrieve payment related information based on a customer ID and a product ID.

#### Features
- Retrieve payment related information by customer ID and product ID for invoicing.
- Serve account data in JSON format.
- Error handling for non-existent customer IDs or product IDs.

#### Details

##### Dependencies
- [Chi](https://github.com/go-chi/chi/v5)

##### Account Struct
The `Account` struct represents account data with the following fields:
- `ProductDescription`: Description of the product.
- `Quantity`: Quantity of the product.
- `UnitPrice`: Unit price of the product.
- `Price`: Price of the product.
- `SubTotal`: Subtotal amount.
- `Tax`: Tax applied.
- `TaxAmount`: Amount of tax.
- `GrandTotal`: Grand total amount.
- `Currency`: Currency of the amount.
- `CurrencySymbol`: Currency symbol.

##### Data Store
Account data is stored in an in memory nested map called `accountData`, where the first level of keys represents customer IDs, and the second level represents product IDs. Each product ID corresponds to an `Account` struct containing the account's information.

##### API Routes
1. **GET /api/accounts/{customerID}/{productID}**: Retrieves account information based on the provided customer ID and product ID.
   - If both the customer ID and product ID exist in the `accountData` map, the corresponding account information is returned in JSON format.
   - If either the customer ID or product ID does not exist, a 404 Not Found error is returned.

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
1. Retrieve account information by customer ID and product ID:
   ```
   GET /api/accounts/CUSTOMER-0001/PRD-160
   ```

#### Summary
The Account API provides an in memory dummy simple service to retrieve payment related information using a RESTful interface. It's a dummy service here all the calculation for the customer will happen for invoicing. It uses the Chi router for routing and handling HTTP requests.
