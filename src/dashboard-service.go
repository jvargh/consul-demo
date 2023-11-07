// https://github.com/hashicorp/demo-consul-101/blob/main/services/dashboard-service/main.go

package main

import (
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	gosocketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
)

var countingServiceURL string
var port string

func main() {
	port = getEnvOrDefault("PORT", "80")
	portWithColon := fmt.Sprintf(":%s", port)

	countingServiceURL = getEnvOrDefault("COUNTING_SERVICE_URL", "http://localhost:9001")

	fmt.Printf("Starting server on http://0.0.0.0:%s\n", port)
	fmt.Println("(Pass as PORT environment variable)")
	fmt.Printf("Using counting service at %s\n", countingServiceURL)
	fmt.Println("(Pass as COUNTING_SERVICE_URL environment variable)")

	failTrack := new(failureTracker)

	router := mux.NewRouter()
	router.PathPrefix("/socket.io/").Handler(startWebsocket(failTrack))
	router.HandleFunc("/health", HealthHandler)
	router.HandleFunc("/health/api", HealthAPIHandler(failTrack))
	router.Handle("/metrics", expvar.Handler())
	router.PathPrefix("/").Handler(http.FileServer(rice.MustFindBox("assets").HTTPBox()))

	log.Fatal(http.ListenAndServe(portWithColon, router))
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// HealthHandler returns a successful status and a message.
// For use by Consul or other processes that need to verify service health.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, you've hit %s\n", r.URL.Path)
}

type failureTracker struct {
	lock     sync.RWMutex
	latest   bool // indicates condition of most recent connection attempt
	failures int  // counts the number of consecutive connection failures
}

func (ft *failureTracker) Count(ok bool) {
	ft.lock.Lock()
	defer ft.lock.Unlock()

	if ft.latest = ok; ok {
		ft.failures = 0
	} else {
		ft.failures++
	}
}

func (ft *failureTracker) Status() (bool, int) {
	ft.lock.RLock()
	defer ft.lock.RUnlock()
	return ft.latest, ft.failures
}

// HealthAPIHandler returns the condition of the connectivity between the
// dashboard and the backend API server.
func HealthAPIHandler(ft *failureTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusOK, failures := ft.Status()
		if statusOK {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, fmt.Sprintf(
				"failures: %d", failures,
			))
		}
	}
}

func startWebsocket(ft *failureTracker) *gosocketio.Server {
	server := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())

	fmt.Println("Starting websocket server...")
	server.On(gosocketio.OnConnection, handleConnectionFunc(ft))
	server.On("send", handleSendFunc(ft))

	return server
}

func handleConnectionFunc(ft *failureTracker) func(c *gosocketio.Channel) {
	return func(c *gosocketio.Channel) {
		fmt.Println("New client connected")
		c.Join("visits")
		handleSendFunc(ft)(c, Count{})
	}
}

func handleSendFunc(ft *failureTracker) func(*gosocketio.Channel, Count) string {
	return func(c *gosocketio.Channel, msg Count) string {
		count, err := getAndParseCount()
		if err != nil {
			count = Count{Count: -1, Message: err.Error(), Hostname: "[Unreachable]"}
			ft.Count(false)
		}
		fmt.Println("Fetched count", count.Count)
		c.Ack("message", count, time.Second*10)
		ft.Count(true)
		return "OK"
	}
}

// Count stores a number that is being counted and other data to send to
// websocket clients.
type Count struct {
	Count    int    `json:"count"`
	Message  string `json:"message"`
	Hostname string `json:"hostname"`
}

func getAndParseCount() (Count, error) {
	url := countingServiceURL

	// NOTE: We use short timeouts so round robin load balancing
	// of services can be experienced during lab demos.
	tr := &http.Transport{
		IdleConnTimeout: time.Second * 1,
	}

	httpClient := http.Client{
		Timeout:   time.Second * 2,
		Transport: tr,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Count{}, err
	}

	req.Header.Set("User-Agent", "HashiCorp Training Lab")

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return Count{}, getErr
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return Count{}, readErr
	}

	defer res.Body.Close()
	return parseCount(body)
}

func parseCount(body []byte) (Count, error) {
	textBytes := []byte(body)

	count := Count{}
	err := json.Unmarshal(textBytes, &count)
	if err != nil {
		fmt.Println(err)
		return count, err
	}
	return count, err
}

// In summary, this code snippet sets up an HTTP and WebSocket server, defines several endpoints and WebSocket events,
// fetches data from an external counting service, and provides some health checking and failure tracking functionality.
// Go code snippet appears to be setting up an HTTP and WebSocket server with some health checking and metrics exposure functionality. 

//Here is a breakdown of its main components and their functions:
// Import Statements:
// ==================
// Libraries for handling HTTP requests, JSON encoding/decoding, logging, and others are imported.
// External libraries such as gorilla/mux for routing, golang-socketio for WebSocket handling, and go.rice for embedding static files are also imported.
// Global Variables:
//==================
// countingServiceURL and port are declared as global variables to hold the URL of the counting service and the port number on which the server will run.
// Main Function:
//==================
// The main() function initializes some configurations like the port number and the counting service URL from environment variables or defaults.
// It sets up a new router using the gorilla/mux library, defines various HTTP handlers, and starts the HTTP server.
// HTTP Handlers:
// ==================
// HealthHandler: Responds with a "Hello, you've hit {URL Path}" message, primarily for checking service health.
// HealthAPIHandler: Checks the connectivity status tracked by a failureTracker instance and responds accordingly.
// startWebsocket: Initializes a WebSocket server and sets up handlers for new connections and a custom "send" event.
// WebSocket Event Handlers:
//==================
// handleConnectionFunc: Handles new client connections to the WebSocket server.
// handleSendFunc: Handles custom "send" events, fetches and sends back a count from an external service.
// Failure Tracking:
//==================
// failureTracker is a custom type with methods to track the success or failure of connection attempts to the external counting service.
// Count Fetching and Parsing:
//==================
// getAndParseCount sends an HTTP request to the external counting service, reads the response, and parses it into a Count struct.
// parseCount function is used to unmarshal the JSON response into a Count struct.
// Utility Function:
//==================
// getEnvOrDefault: A utility function to get the value of an environment variable or return a default value if the environment variable is not set.
// Count Struct:
//==================
// A struct to hold the count data along with some other information to be sent to WebSocket clients.
