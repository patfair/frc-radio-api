package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
)

const port = 8081

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request from %s for %s with method %s", r.RemoteAddr, r.URL.Path, r.Method)
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	ipAddress, err := getVlan100IpAddress()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	listenAddress := fmt.Sprintf("%s:%d", ipAddress, port)
	http.HandleFunc("/", handler)
	fmt.Printf("Server listening on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatal(err)
	}
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
