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

// runWebServer starts the HTTP server and blocks until the process terminates, serving requests.
func runWebServer() {
	ipAddress, err := getVlan100IpAddress()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	listenAddress := fmt.Sprintf("%s:%d", ipAddress, port)
	log.Printf("Server listening on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, newRouter()); err != nil {
		log.Fatal(err)
	}
}

// newRouter sets up the mapping between URLs and handlers.
func newRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/", rootHandler).Methods("GET")
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/status", statusHandler).Methods("GET")
	return router
}

// rootHandler redirects the root URL to the status page.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/status", http.StatusFound)
}

// healthHandler returns a simple "OK" response to indicate that the server is running.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "OK")
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
