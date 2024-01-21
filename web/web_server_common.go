package web

import (
	"filippo.io/age"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/patfair/frc-radio-api/radio"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	// TCP port that the web server listens on.
	port = 80

	// Path to the optional file containing the password for the API.
	passwordFilePath = "/root/frc-radio-api-password.txt"

	// Interval between attempts to get the IP address of the radio on startup.
	ipAddressPollIntervalSec = 3
)

// WebServer holds shared state across requests to the API.
type WebServer struct {
	// Password for authorizing requests to the API. If blank, no authorization is required.
	password string

	// Private key for decrypting new firmware. If nil, only unencrypted firmware can be uploaded.
	firmwareDecryptionKey *age.X25519Identity

	// Device that the API provides access to.
	radio *radio.Radio
}

// NewWebServer creates a new server instance.
func NewWebServer(radio *radio.Radio) *WebServer {
	return &WebServer{radio: radio}
}

// Run starts the HTTP server and blocks until the process terminates, serving requests.
func (web *WebServer) Run() {
	web.setUpSecrets()

	listenAddress := getListenAddress()
	log.Printf("Server listening on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, web.newRouter()); err != nil {
		log.Fatal(err)
	}
}

// setUpSecrets reads the password and firmware decryption keys from their respective files, if they exist.
func (web *WebServer) setUpSecrets() {
	passwordBytes, err := os.ReadFile(passwordFilePath)
	if err != nil {
		log.Printf("Error opening password file; authorization disabled: %v", err)
	} else {
		web.password = strings.TrimSpace(string(passwordBytes))
	}

	privateKeyBytes, err := os.ReadFile(firmwareDecryptionKeyFilePath)
	if err != nil {
		log.Printf("Error opening encryption key file; firmware decryption disabled: %v", err)
	} else if len(privateKeyBytes) != 0 {
		privateKey := strings.TrimSpace(string(privateKeyBytes))
		web.firmwareDecryptionKey, err = age.ParseX25519Identity(privateKey)
		if err != nil {
			log.Printf("Error parsing encryption key; firmware decryption disabled: %v", err)
		}
	}
}

// newRouter sets up the mapping between URLs and handlers.
func (web *WebServer) newRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/", web.rootHandler).Methods("GET")
	router.HandleFunc("/health", web.healthHandler).Methods("GET")
	router.HandleFunc("/status", web.statusHandler).Methods("GET")
	router.HandleFunc("/configuration", web.configurationHandler).Methods("POST")
	router.HandleFunc("/firmware", web.firmwareHandler).Methods("POST")
	addRoutes(router, web)
	return router
}

// rootHandler redirects the root URL to the status page.
func (web *WebServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/configuration", http.StatusFound)
}

// healthHandler returns a simple "OK" response to indicate that the server is running.
func (web *WebServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, "OK")
}

// isAuthorized returns true if the request is authorized to access the API.
func (web *WebServer) isAuthorized(r *http.Request) bool {
	if web.password == "" {
		return true
	}
	var password string
	_, _ = fmt.Sscanf(r.Header.Get("Authorization"), "Bearer %s", &password)
	return password == web.password
}

// handleWebErr writes the given error out as plain text with the given status code.
func handleWebErr(w http.ResponseWriter, err error, statusCode int) {
	message := fmt.Sprintf("HTTP request error %d: %v", statusCode, err)
	log.Println(message)
	http.Error(w, message, statusCode)
}
