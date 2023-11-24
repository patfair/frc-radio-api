package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

// getHttpResponse stubs the webserver, sends a GET request to the given path, and returns the response, for use in
// testing.
func (web *WebServer) getHttpResponse(path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}

// postHttpResponse stubs the webserver, sends a POST request to the given path with the given body, and returns the
// response, for use in testing.
func (web *WebServer) postHttpResponse(path string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}
