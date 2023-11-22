package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeb_rootHandler(t *testing.T) {
	var web web
	recorder := web.getHttpResponse("/")
	assert.Equal(t, 302, recorder.Code)
	assert.Equal(t, recorder.Header().Get("Location"), "/status")
}

func TestWeb_healthHandler(t *testing.T) {
	var web web
	recorder := web.getHttpResponse("/health")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, recorder.Body.String(), "OK\n")
}

func TestWebNotFound(t *testing.T) {
	var web web
	recorder := web.getHttpResponse("/foo")
	assert.Equal(t, 404, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "404 page not found")
}

func TestGetVlan100IpAddress(t *testing.T) {
	ipAddress, err := getVlan100IpAddress()

	// Branch the test verification logic since it may or may not be run on a system with a 10.0.100.x interface and
	// mocking the system calls to be deterministic is onerous.
	if err == nil {
		assert.Regexp(t, "^10\\.0\\.100\\.\\d+$", ipAddress)
	} else {
		assert.Contains(t, err.Error(), "no IP address found on VLAN 100")
	}
}

// getHttpResponse stubs the webserver, sends a GET request to the given path, and returns the response, for use in
// testing.
func (web *web) getHttpResponse(path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}

// postHttpResponse stubs the webserver, sends a POST request to the given path with the given body, and returns the
// response, for use in testing.
func (web *web) postHttpResponse(path string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}
