package web

import (
	"encoding/json"
	"net/http"
)

// statusHandler returns a JSON dump of the access point status.
func (web *WebServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	jsonData, err := json.MarshalIndent(web.accessPoint, "", "  ")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}
