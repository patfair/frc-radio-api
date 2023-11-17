package main

import (
	"fmt"
	"log"
	"net/http"
)

const port = 8081

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request from %s for %s with method %s", r.RemoteAddr, r.URL.Path, r.Method)
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Printf("Server listening on :%d\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
