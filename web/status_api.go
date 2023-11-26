package web

import (
	"encoding/json"
	"net/http"
)

// statusHandler returns a JSON dump of the radio status.
func (web *WebServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		http.Error(
			w, "Not authorized; must provide 'Authorization: Bearer [password]' header.", http.StatusUnauthorized,
		)
		return
	}

	jsonData, err := json.MarshalIndent(web.radio, "", "  ")
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
