package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := 3000
	flag.IntVar(&port,"p",port,"port to listen on")
	flag.Parse()

	htmxPath := os.Getenv("HTMX_SRC")

	if htmxPath == "" {
		log.Fatal("HTMX_SRC environment variable should be set to a path")
	}
	if _, err := os.Stat(htmxPath); os.IsNotExist(err) {
		log.Fatalf("HTMX_SRC environment variable points to a file that does not exist (%s)", htmxPath)
	}

	log.Printf("HTMX file: %s", htmxPath)

	http.HandleFunc("/static/js/htmx.min.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, htmxPath)
	})

	log.Printf("Server starting on :%d",port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d",port), nil); err != nil {
		log.Fatal(err)
	}
}


