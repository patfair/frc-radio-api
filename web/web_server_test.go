package web

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWeb_rootHandler(t *testing.T) {
	var web WebServer
	recorder := web.getHttpResponse("/")
	assert.Equal(t, 302, recorder.Code)
	assert.Equal(t, recorder.Header().Get("Location"), "/status")
}

func TestWeb_healthHandler(t *testing.T) {
	var web WebServer
	recorder := web.getHttpResponse("/health")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, recorder.Body.String(), "OK\n")
}

func TestWebNotFound(t *testing.T) {
	var web WebServer
	recorder := web.getHttpResponse("/foo")
	assert.Equal(t, 404, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "404 page not found")
}
