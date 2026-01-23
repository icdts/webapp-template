package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"os"
)

func main() {
	var err error
	var port int 

	portStr := os.Getenv("PORT")
	if port, err = strconv.Atoi(portStr); err != nil {
		log.Fatalf("PORT couldn't be converted to an int: %s", portStr)
	}

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "views/index.html")
	})

	log.Printf("Server starting on :%d",port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d",port), nil); err != nil {
		log.Fatal(err)
	}
}


