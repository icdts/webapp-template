package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
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

	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/readyz", readyz)

	http.HandleFunc("/assets/htmx.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, htmxPath) })
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", pageIndex)
	http.HandleFunc("/time", pageTime)

	log.Printf("Server starting on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal(err)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func readyz(w http.ResponseWriter, _ *http.Request) {
	// TODO: In real app, you need to check database and/or cache
	// if db.Ping() != nil { http.Error(w,"DB Not Ready", 503); return }
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func pageIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("views/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func pageTime(w http.ResponseWriter, r *http.Request) {
	ts := time.Now().Format(time.RFC1123)
	fmt.Fprintf(w, `
			<button hx-get="/time" hx-swap="outerHTML" style="background-color: #d1fae5; border: 1px solid green; padding: 10px; border-radius: 5px;">
				Verified at: %s
			</button>
	`, ts)
}
