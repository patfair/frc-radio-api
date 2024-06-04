// This file is specific to the robot radio version of the API.
//go:build robot

package web

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/patfair/frc-radio-api/radio"
	"net/http"
)

// TCP port that the web server listens on.
const port = 80

// getListenAddress returns the address and port that the web server should listen on.
func getListenAddress(r *radio.Radio) string {
	return fmt.Sprintf(":%d", port)
}

// addRoutes adds additional route handlers to the router if needed.
func addRoutes(router *mux.Router, web *WebServer) {
	router.HandleFunc("/configuration", web.configurationPageHandler).Methods("GET")
}

// rootHandler redirects the root URL to the configuration page.
func (web *WebServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/configuration", http.StatusFound)
}
