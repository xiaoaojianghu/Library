package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type BookDetail struct {
	Title           string `json:"title"`
	AvailableCopies int    `json:"availableCopies"`
}

type LoanDetail struct {
	BookTitle      string    `json:"bookTitle"`
	NameOfBorrower string    `json:"nameOfBorrower"`
	LoanDate       time.Time `json:"loanDate"`
	ReturnDate     time.Time `json:"returnDate"`
}

type Library struct {
	Books map[string]BookDetail
	Loans map[string][]LoanDetail
	mutex sync.RWMutex
}

func NewLibrary() *Library {
	lib := &Library{
		Books: make(map[string]BookDetail),
		Loans: make(map[string][]LoanDetail),
	}

	lib.Books["Go Programming"] = BookDetail{Title: "Go Programming", AvailableCopies: 3}
	lib.Books["Clean Code"] = BookDetail{Title: "Clean Code", AvailableCopies: 2}

	return lib
}

func main() {
	library := NewLibrary()

	http.HandleFunc("/Book", library.getBookHandler)
	http.HandleFunc("/Borrow", library.borrowBookHandler)
	http.HandleFunc("/Extend", library.extendLoanHandler)
	http.HandleFunc("/Return", library.returnBookHandler)

	fmt.Println("Starting e-Library server on :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func (l *Library) getBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Title query parameter is required", http.StatusBadRequest)
		return
	}

	l.mutex.RLock()
	book, exists := l.Books[title]
	l.mutex.RUnlock()

	if !exists {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func (l *Library) borrowBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Title    string `json:"title"`
		Borrower string `json:"borrower"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Title == "" || request.Borrower == "" {
		http.Error(w, "Title and borrower are required", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	book, exists := l.Books[request.Title]
	if !exists {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	if book.AvailableCopies <= 0 {
		http.Error(w, "No copies available", http.StatusConflict)
		return
	}

	book.AvailableCopies--
	l.Books[request.Title] = book

	now := time.Now()
	loan := LoanDetail{
		BookTitle:      request.Title,
		NameOfBorrower: request.Borrower,
		LoanDate:       now,
		ReturnDate:     now.AddDate(0, 0, 28), // 4 weeks loan period
	}

	l.Loans[request.Title] = append(l.Loans[request.Title], loan)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loan)
}

func (l *Library) extendLoanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Title    string `json:"title"`
		Borrower string `json:"borrower"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Title == "" || request.Borrower == "" {
		http.Error(w, "Title and borrower are required", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	loans, exists := l.Loans[request.Title]
	if !exists {
		http.Error(w, "No loans found for this book", http.StatusNotFound)
		return
	}

	var loanFound bool
	var extendedLoan LoanDetail

	for i, loan := range loans {
		if loan.NameOfBorrower == request.Borrower {
			// Extend loan by 3 weeks from current return date
			loans[i].ReturnDate = loan.ReturnDate.AddDate(0, 0, 21)
			extendedLoan = loans[i]
			loanFound = true
			break
		}
	}

	if !loanFound {
		http.Error(w, "No loan found for this borrower", http.StatusNotFound)
		return
	}

	l.Loans[request.Title] = loans

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(extendedLoan)
}

func (l *Library) returnBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Title    string `json:"title"`
		Borrower string `json:"borrower"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Title == "" || request.Borrower == "" {
		http.Error(w, "Title and borrower are required", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	loans, exists := l.Loans[request.Title]
	if !exists {
		http.Error(w, "No loans found for this book", http.StatusNotFound)
		return
	}

	book, bookExists := l.Books[request.Title]
	if !bookExists {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	var loanIndex = -1
	for i, loan := range loans {
		if loan.NameOfBorrower == request.Borrower {
			loanIndex = i
			break
		}
	}

	if loanIndex == -1 {
		http.Error(w, "No loan found for this borrower", http.StatusNotFound)
		return
	}

	// Remove the loan by swapping with the last element and truncating
	loans[loanIndex] = loans[len(loans)-1]
	l.Loans[request.Title] = loans[:len(loans)-1]

	book.AvailableCopies++
	l.Books[request.Title] = book

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Book '%s' successfully returned by %s", request.Title, request.Borrower),
	})
}
