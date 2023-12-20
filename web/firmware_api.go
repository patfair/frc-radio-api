package web

import (
	"net/http"
)

// firmwareHandler handles requests to update the radio firmware.
func (web *WebServer) firmwareHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		http.Error(
			w, "Not authorized; must provide 'Authorization: Bearer [password]' header.", http.StatusUnauthorized,
		)
		return
	}

}
