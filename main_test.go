package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetBookHandler(t *testing.T) {
	library := NewLibrary()

	// Test 1: Get an existing book
	req, err := http.NewRequest("GET", "/Book?title=Go Programming", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(library.getBookHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var book BookDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &book); err != nil {
		t.Fatal(err)
	}

	if book.Title != "Go Programming" {
		t.Errorf("expected book title 'Go Programming', got '%s'", book.Title)
	}

	// Test 2: Get a non-existent book
	req, err = http.NewRequest("GET", "/Book?title=Nonexistent Book", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestBorrowBookHandler(t *testing.T) {
	library := NewLibrary()

	requestBody := map[string]string{
		"title":    "Go Programming",
		"borrower": "John Doe",
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	// Test borrowing a book
	req, err := http.NewRequest("POST", "/Borrow", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(library.borrowBookHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var loan LoanDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &loan); err != nil {
		t.Fatal(err)
	}

	if loan.BookTitle != "Go Programming" || loan.NameOfBorrower != "John Doe" {
		t.Errorf("unexpected loan details: got title '%s' and borrower '%s'", loan.BookTitle, loan.NameOfBorrower)
	}

	// Verify book copies were reduced
	library.mutex.RLock()
	book := library.Books["Go Programming"]
	library.mutex.RUnlock()
	if book.AvailableCopies != 2 {
		t.Errorf("expected 2 available copies, got %d", book.AvailableCopies)
	}
}

func TestExtendLoanHandler(t *testing.T) {
	library := NewLibrary()

	// First, create a loan to extend
	now := time.Now()
	originalReturnDate := now.AddDate(0, 0, 28) // 4 weeks
	loan := LoanDetail{
		BookTitle:      "Clean Code",
		NameOfBorrower: "Jane Smith",
		LoanDate:       now,
		ReturnDate:     originalReturnDate,
	}

	library.mutex.Lock()
	library.Books["Clean Code"] = BookDetail{Title: "Clean Code", AvailableCopies: 1} // One is borrowed
	library.Loans["Clean Code"] = []LoanDetail{loan}
	library.mutex.Unlock()

	// Prepare request body for extension
	requestBody := map[string]string{
		"title":    "Clean Code",
		"borrower": "Jane Smith",
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	// Test extending the loan
	req, err := http.NewRequest("POST", "/Extend", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(library.extendLoanHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var extendedLoan LoanDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &extendedLoan); err != nil {
		t.Fatal(err)
	}

	expectedNewReturnDate := originalReturnDate.AddDate(0, 0, 21) // 3 more weeks
	if extendedLoan.ReturnDate.Format("2006-01-02") != expectedNewReturnDate.Format("2006-01-02") {
		t.Errorf("unexpected return date: got %v, expected %v",
			extendedLoan.ReturnDate.Format("2006-01-02"),
			expectedNewReturnDate.Format("2006-01-02"))
	}
}

func TestReturnBookHandler(t *testing.T) {
	library := NewLibrary()

	// First, create a loan to return
	loan := LoanDetail{
		BookTitle:      "Design Patterns",
		NameOfBorrower: "Bob Johnson",
		LoanDate:       time.Now(),
		ReturnDate:     time.Now().AddDate(0, 0, 28),
	}

	library.mutex.Lock()
	library.Books["Design Patterns"] = BookDetail{Title: "Design Patterns", AvailableCopies: 0}
	library.Loans["Design Patterns"] = []LoanDetail{loan}
	library.mutex.Unlock()

	// Prepare request body for returning
	requestBody := map[string]string{
		"title":    "Design Patterns",
		"borrower": "Bob Johnson",
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	// Test returning the book
	req, err := http.NewRequest("POST", "/Return", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(library.returnBookHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify book copies were increased
	library.mutex.RLock()
	book := library.Books["Design Patterns"]
	library.mutex.RUnlock()
	if book.AvailableCopies != 1 {
		t.Errorf("expected 1 available copy, got %d", book.AvailableCopies)
	}

	// Verify loan was removed
	library.mutex.RLock()
	loans := library.Loans["Design Patterns"]
	library.mutex.RUnlock()
	if len(loans) != 0 {
		t.Errorf("expected loan to be removed, but found %d loans", len(loans))
	}
}
