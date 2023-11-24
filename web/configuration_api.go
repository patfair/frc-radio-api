package web

import (
	"encoding/json"
	"fmt"
	"github.com/patfair/frc-radio-api/radio"
	"log"
	"net/http"
)

// configurationHandler receives a JSON request to configure the radio and adds it to the asynchronous queue.
func (web *WebServer) configurationHandler(w http.ResponseWriter, r *http.Request) {
	var request radio.ConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		errorMessage := "Error: invalid JSON: " + err.Error()
		log.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	if err := request.Validate(web.radio.Type); err != nil {
		errorMessage := "Error: " + err.Error()
		log.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	log.Printf("Received configuration request: %+v", request)
	web.radio.ConfigurationRequestChannel <- request
	_, _ = fmt.Fprintln(w, "New configuration received and will be applied asynchronously.")
}
