package web

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/patfair/frc-radio-api/radio"
	"log"
	"net/http"
)

const port = 8081

// WebServer holds shared state across requests to the API.
type WebServer struct {
	radio *radio.Radio
}

// NewWebServer creates a new server instance.
func NewWebServer(radio *radio.Radio) *WebServer {
	return &WebServer{radio: radio}
}

// Run starts the HTTP server and blocks until the process terminates, serving requests.
func (web *WebServer) Run() {
	ipAddress, err := getVlan100IpAddress()
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	listenAddress := fmt.Sprintf("%s:%d", ipAddress, port)
	log.Printf("Server listening on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, web.newRouter()); err != nil {
		log.Fatal(err)
	}
}

// newRouter sets up the mapping between URLs and handlers.
func (web *WebServer) newRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/", web.rootHandler).Methods("GET")
	router.HandleFunc("/health", web.healthHandler).Methods("GET")
	router.HandleFunc("/status", web.statusHandler).Methods("GET")
	router.HandleFunc("/configuration", web.configurationHandler).Methods("POST")
	return router
}

// rootHandler redirects the root URL to the status page.
func (web *WebServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/status", http.StatusFound)
}

// healthHandler returns a simple "OK" response to indicate that the server is running.
func (web *WebServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, "OK")
}
