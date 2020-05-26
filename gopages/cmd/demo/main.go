package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	fs := http.Dir("dist")
	fileServer := http.FileServer(fs)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fs.Open(r.URL.Path)
		if err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".go") {
			// GitHub Pages automatically looks up a corresponding .html file if it exists
			_, err := fs.Open(r.URL.Path + ".html")
			if err == nil {
				r.URL.Path += ".html"
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/404.html"
		r.URL.RawPath = "/404.html"
		fileServer.ServeHTTP(w, r)
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	fmt.Println("Starting demo server on :8080...")
	_ = server.ListenAndServe()
}
