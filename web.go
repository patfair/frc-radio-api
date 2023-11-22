package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"regexp"
)

const port = 8081

// web holds shared state across requests to the web server.
type web struct {
	accessPoint *accessPoint
}

// newWeb creates a new web instance.
func newWeb(accessPoint *accessPoint) *web {
	return &web{accessPoint: accessPoint}
}

// run starts the HTTP server and blocks until the process terminates, serving requests.
func (web *web) run() {
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
func (web *web) newRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/", web.rootHandler).Methods("GET")
	router.HandleFunc("/health", web.healthHandler).Methods("GET")
	router.HandleFunc("/status", web.statusHandler).Methods("GET")
	router.HandleFunc("/configuration", web.configurationHandler).Methods("POST")
	return router
}

// rootHandler redirects the root URL to the status page.
func (web *web) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/status", http.StatusFound)
}

// healthHandler returns a simple "OK" response to indicate that the server is running.
func (web *web) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, "OK")
}

// handleWebErr writes the given error out as plain text with a status code of 500.
func handleWebErr(w http.ResponseWriter, err error) {
	log.Printf("HTTP request error: %v", err)
	http.Error(w, "Internal server error: "+err.Error(), 500)
}

// getVlan100IpAddress returns the IP address of the first interface that has an IP address on the 10.0.100.x VLAN.
func getVlan100IpAddress() (string, error) {
	ipRe := regexp.MustCompile("^(10\\.0\\.100\\.\\d+)")
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			address := addr.String()
			matches := ipRe.FindStringSubmatch(address)
			if len(matches) != 0 {
				return matches[1], nil
			}
		}
	}
	return "", fmt.Errorf("no IP address found on VLAN 100 (i.e. matching %v)", ipRe)
}
