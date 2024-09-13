// This file is specific to the access point version of the API.
//go:build !robot

package web

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const (
	scanScriptPath = "/root/scan.sh"
	scanOutputPath = "/root/scan_output.txt"
)

func (web *WebServer) startScanHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		handleWebErr(
			w,
			errors.New("not authorized; must provide 'Authorization: Bearer [password]' header"),
			http.StatusUnauthorized,
		)
		return
	}

	cmd := exec.Command(scanScriptPath, scanOutputPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get stdout for scan: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "Failed to start scan")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start scan.sh: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "Failed to start scan")
		return
	}

	// Wait to make sure it doesn't become a zombie process
	go cmd.Wait()

	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read scan.sh output: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "Failed to start scan")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, line)
}

func (web *WebServer) scanResultHandler(w http.ResponseWriter, r *http.Request) {
	if !web.isAuthorized(r) {
		handleWebErr(
			w,
			errors.New("not authorized; must provide 'Authorization: Bearer [password]' header"),
			http.StatusUnauthorized,
		)
		return
	}

	w.WriteHeader(http.StatusOK)

	bytes, err := os.ReadFile(scanOutputPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	result := string(bytes)

	_, _ = fmt.Fprintln(w, result)
}
