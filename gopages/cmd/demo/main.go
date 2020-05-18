package main

import (
	"fmt"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("dist"))

	server := http.Server{
		Addr:    ":8080",
		Handler: fs,
	}
	fmt.Println("Starting demo server on :8080...")
	server.ListenAndServe()
}
