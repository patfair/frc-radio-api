package web

import (
	"bytes"
	"io"
	"mime/multipart"
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

// getHttpResponseWithHeaders stubs the webserver, sends a GET request to the given path with the given headers, and
// returns the response, for use in testing.
func (web *WebServer) getHttpResponseWithHeaders(path string, headers map[string]string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
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

// postHttpResponseWithHeaders stubs the webserver, sends a POST request to the given path with the given body and the
// given headers, and returns the response, for use in testing.
func (web *WebServer) postHttpResponseWithHeaders(
	path string, body string, headers map[string]string,
) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}

// postFileHttpResponse stubs the webserver, sends a POST request to the given path with the given file and other
// fields, and returns the response, for use in testing.
func (web *WebServer) postFileHttpResponse(
	path string, paramName string, fileContent []byte, otherFields map[string]string,
) *httptest.ResponseRecorder {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(paramName, "file.ext")
	io.Copy(part, bytes.NewReader(fileContent))
	for key, value := range otherFields {
		writer.WriteField(key, value)
	}
	writer.Close()
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	web.newRouter().ServeHTTP(recorder, req)
	return recorder
}
