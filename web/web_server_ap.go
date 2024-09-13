// This file is specific to the access point version of the API.
//go:build !robot

package web

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/patfair/frc-radio-api/radio"
	"log"
	"net"
	"net/http"
	"regexp"
	"time"
)

const (
	// TCP port that the web server listens on.
	portLinksys      = 8081
	portVividHosting = 80
)

// getListenAddress returns the address and port that the web server should listen on.
func getListenAddress(r *radio.Radio) string {
	var port int
	if r.Type == radio.TypeLinksys {
		port = portLinksys
	} else {
		port = portVividHosting
	}

	var ipAddress string
	for {
		var err error
		ipAddress, err = getVlan100IpAddress()
		if err != nil {
			log.Printf("Error getting radio IP address; trying again later: %v", err)
			time.Sleep(ipAddressPollIntervalSec * time.Second)
			continue
		}
		break
	}

	return fmt.Sprintf("%s:%d", ipAddress, port)
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

// addRoutes adds additional route handlers to the router if needed.
func addRoutes(router *mux.Router, web *WebServer) {
	router.HandleFunc("/scan/start", web.startScanHandler).Methods("GET")
	router.HandleFunc("/scan/result", web.scanResultHandler).Methods("GET")
}

// rootHandler redirects the root URL to the status page.
func (web *WebServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/status", http.StatusFound)
}
