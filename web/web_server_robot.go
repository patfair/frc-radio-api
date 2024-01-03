// This file is specific to the robot radio version of the API.
//go:build robot

package web

import (
	"fmt"
	"github.com/gorilla/mux"
)

// getListenAddress returns the address and port that the web server should listen on.
func getListenAddress() string {
	return fmt.Sprintf(":%d", port)
}

// addRoutes adds additional route handlers to the router if needed.
func addRoutes(router *mux.Router, web *WebServer) {
	router.HandleFunc("/configuration", web.configurationPageHandler).Methods("GET")
}
