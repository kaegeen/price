package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// Item represents a basic data structure for our API
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// In-memory database of items
var items = []Item{
	{ID: 1, Name: "Item One", Price: 100},
	{ID: 2, Name: "Item Two", Price: 200},
}

// Logging middleware that logs each request to the server
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("Request %s %s - Duration: %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// GetItemsHandler returns all items in JSON format
func GetItemsHandler(w http.ResponseWriter, r *http.Request) {
	// Using context to check for cancellation
	ctx := r.Context()

	// Simulate a long-running request (e.g., database lookup)
	select {
	case <-time.After(2 * time.Second):
		// Convert items to JSON and send as response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	case <-ctx.Done():
		// If request is cancelled, log and return an error
		log.Println("Request was cancelled by the client")
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
	}
}

// AddItemHandler adds a new item (simulating database insert)
func AddItemHandler(w http.ResponseWriter, r *http.Request) {
	var newItem Item
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&newItem); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Append to the in-memory list (simulating a database insert)
	newItem.ID = len(items) + 1
	items = append(items, newItem)

	// Return a success message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newItem)
}

// ServeStaticFiles serves static content (e.g., HTML, CSS, JS)
func ServeStaticFiles(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static"+r.URL.Path)
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func main() {
	// Initialize the router
	r := mux.NewRouter()

	// Middleware setup
	r.Use(loggingMiddleware)

	// Static file serving (example: static/ folder should have an index.html)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// API routes
	r.HandleFunc("/api/items", GetItemsHandler).Methods("GET")
	r.HandleFunc("/api/items", AddItemHandler).Methods("POST")

	// Catch-all for 404s
	r.NotFoundHandler = http.HandlerFunc(NotFoundHandler)

	// Start the HTTP server with context cancellation support
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown handling
	go func() {
		log.Println("Server started on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown: wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	<-quit

	// Cancel all ongoing requests and shutdown the server gracefully
	log.Println("Shutting down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown Failed:", err)
	}
	log.Println("Server exited gracefully")
}
