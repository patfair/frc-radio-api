package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
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
	assert.Equal(t, recorder.Body.String(), "OK")
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

func (web *web) getHttpResponse(path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}
