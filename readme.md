## Components:
1. **BookDetail**: Stores information about a book
2. **LoanDetail**: Represents a book loan with borrower name and dates
3. **Library**: In-memory storage using maps with mutex for thread safety

## Endpoints:

### 1. Get Book Details
- **Endpoint**: `GET /Book?title=<book_title>`
- **Description**: Retrieves details of a specific book
- **Response**: Book details including available copies

### 2. Borrow a Book
- **Endpoint**: `POST /Borrow`
- **Description**: Borrows a book with a 4-week loan period
- **Request Body**:
  ```json
  {
    "title": "Go Programming",
    "borrower": "John Doe"
  }
  ```
- **Response**: Loan details including return date

### 3. Extend a Loan
- **Endpoint**: `POST /Extend`
- **Description**: Extends a loan by 3 weeks from the current return date
- **Request Body**:
  ```json
  {
    "title": "Go Programming",
    "borrower": "John Doe"
  }
  ```
- **Response**: Updated loan details

### 4. Return a Book
- **Endpoint**: `POST /Return`
- **Description**: Returns a borrowed book
- **Request Body**:
  ```json
  {
    "title": "Go Programming",
    "borrower": "John Doe"
  }
  ```
- **Response**: Success message and status
