package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
)

const logFilePath = "/var/log/frc-radio-api.log"
const port = 8081

func main() {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.Printf("Error opening log file; logging to stdout instead: %v", err)
	}

	ipAddress, err := getVlan100IpAddress()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", rootHandler).Methods("GET")
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/status", statusHandler).Methods("GET")

	listenAddress := fmt.Sprintf("%s:%d", ipAddress, port)
	log.Printf("Server listening on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, router); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "OK")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "Status page")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/status", http.StatusFound)
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
