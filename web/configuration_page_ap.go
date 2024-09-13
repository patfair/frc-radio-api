// This file is specific to the access point version of the API.
//go:build !robot

package web

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
)

//go:embed configuration_page_ap.html
var htmlContents string

// configPageHandler receives a GET request and returns the radio configuration html page.
func (web *WebServer) configurationPageHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		handleWebErr(
			w,
			errors.New("not authorized; must provide 'Authorization: Bearer [password]' header"),
			http.StatusUnauthorized,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintln(w, htmlContents)
}
