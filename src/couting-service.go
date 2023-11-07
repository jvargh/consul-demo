// https://github.com/hashicorp/demo-consul-101/blob/main/services/counting-service/main.go

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	portWithColon := fmt.Sprintf(":%s", port)

	router := mux.NewRouter()
	router.HandleFunc("/health", HealthHandler)

	var index uint64
	router.PathPrefix("/").Handler(CountHandler{index: &index})

	// Serve!
	fmt.Printf("Serving at http://localhost:%s\n(Pass as PORT environment variable)\n", port)
	log.Fatal(http.ListenAndServe(portWithColon, router))
}

// HealthHandler returns a succesful status and a message.
// For use by Consul or other processes that need to verify service health.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, you've hit %s\n", r.URL.Path)
}

// Count stores a number that is being counted and other data to
// return as JSON in the API.
type Count struct {
	Count    uint64 `json:"count"`
	Hostname string `json:"hostname"`
}

// CountHandler serves a JSON feed that contains a number that increments each time
// the API is called.
type CountHandler struct {
	index *uint64
}

func (h CountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(h.index, 1)
	hostname, _ := os.Hostname()
	index := atomic.LoadUint64(h.index)

	count := Count{Count: index, Hostname: hostname}

	responseJSON, _ := json.Marshal(count)
	fmt.Fprintf(w, string(responseJSON))
}


// Go code sets up an HTTP server with two endpoints:
//
// Health Check Endpoint (/health):
// ================================
// 1. This endpoint is designed for health checking purposes, which is common in microservice architectures to ensure a 
// service is running and healthy.
// 2. When a request is made to /health, the HealthHandler function is called. 
// This function responds with a HTTP status 200 (OK) and prints a message indicating the path that was hit.
//
// Count Endpoint (/):
// ===================
// 1. This endpoint is set up to increment a counter each time its accessed and return a JSON object with the current count and hostname.
// 2. The count is stored atomically, meaning its thread-safe, ensuring accurate counting even under concurrent access.
// 3. The CountHandler struct holds a pointer to a uint64 which tracks the count of requests. 
//    The ServeHTTP method of this struct is implemented to handle requests made to the root path (/).
// 4. When a request is made, it increments the count using atomic.AddUint64, retrieves the hostname of the machine where the server is running, then constructs a Count object with the current count and hostname.
// 5. This Count object is serialized to JSON and sent back as the response.
//
// The HTTP server is set up using the gorilla/mux package, which is a powerful HTTP router and URL matcher for building Go web servers. 
// The http.ListenAndServe function is called with the address to listen on and the router object, which dispatches incoming requests 
// to the appropriate handler functions based on the URL path.
//
// The main function:
// ==================
// 1. First, it tries to get the port number from the PORT environment variable, and defaults to port 80 if the environment variable isnt set.
// 2. It creates a new router, sets up the two endpoints as described above, then starts the server.
// 3. It also prints a message to the console indicating the address where the server is listening.
