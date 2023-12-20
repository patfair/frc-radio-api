package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/patfair/frc-radio-api/radio"
	"log"
	"net/http"
)

// configurationHandler receives a JSON request to configure the radio and adds it to the asynchronous queue.
func (web *WebServer) configurationHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		handleWebErr(
			w,
			errors.New("not authorized; must provide 'Authorization: Bearer [password]' header"),
			http.StatusUnauthorized,
		)
		return
	}

	var request radio.ConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		handleWebErr(w, fmt.Errorf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if err := request.Validate(web.radio); err != nil {
		handleWebErr(w, fmt.Errorf("invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Received configuration request: %+v", request)
	web.radio.ConfigurationRequestChannel <- request
	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintln(w, "New configuration received and will be applied asynchronously.")
}
