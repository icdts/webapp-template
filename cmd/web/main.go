package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/icdts/webapp/internal/db"
	"github.com/icdts/webapp/internal/models"

	"github.com/jmoiron/sqlx"
)

// AppData holds data passed to templates
type AppData struct {
	Items []models.Item
}

type HandleFuncWithDb func(*sqlx.DB, http.ResponseWriter, *http.Request)

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

	dbPath := "tmp/app.db"
	database, err := db.Connect(dbPath)
	if err != nil {
		log.Fatal("Failed to init DB:", err)
	}
	defer database.Close()

	setupRoutes(htmxPath, database)

	log.Printf("Server starting on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal(err)
	}
}

func withDatabase(d *sqlx.DB, f HandleFuncWithDb) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(d, w, r)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func readyz(database *sqlx.DB, w http.ResponseWriter, _ *http.Request) {
	if database.Ping() != nil {
		http.Error(w, "DB Not Ready", 503)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func setupRoutes(htmxPath string, database *sqlx.DB) {
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/readyz", withDatabase(database, readyz))

	http.HandleFunc("/assets/htmx.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, htmxPath) })
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", withDatabase(database, pageIndex))
	http.HandleFunc("/time", pageTime)
}

func pageIndex(database *sqlx.DB, w http.ResponseWriter, r *http.Request) {
	items := []models.Item{}
	// Select directly into the struct slice
	err := database.Select(&items, "SELECT * FROM items ORDER BY id ASC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	tmpl, err := template.ParseFiles("views/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := AppData{Items: items}
	tmpl.Execute(w, data)
}

func pageTime(w http.ResponseWriter, r *http.Request) {
	ts := time.Now().Format(time.RFC1123)
	fmt.Fprintf(w, `
			<button hx-get="/time" hx-swap="outerHTML" style="background-color: #d1fae5; border: 1px solid green; padding: 10px; border-radius: 5px;">
				Verified at: %s
			</button>
	`, ts)
}
