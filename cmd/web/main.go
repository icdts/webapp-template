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

type App struct {
	SQLite   *sqlx.DB
	Postgres *sqlx.DB
}

type AppData struct {
	Items  []models.Item
	Source string
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

	sqDB, err := db.ConnectSQLite("tmp/app.db")
	if err != nil {
		log.Fatal("Failed to init sqlite DB:", err)
	}
	defer sqDB.Close()

	pgDSN := os.Getenv("DATABASE_URL")
	pgDB, err := db.ConnectPostgres(pgDSN)
	if err != nil {
		log.Println("Failed to init Postgres DB:", err)
	}else{
		defer pgDB.Close()
	}

	app := &App{
		SQLite:   sqDB,
		Postgres: pgDB,
	}

	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/readyz", app.readyz)
	http.HandleFunc("/assets/htmx.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, htmxPath) })
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", app.pageIndex)
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

func (app App) readyz(w http.ResponseWriter, _ *http.Request) {
	if app.SQLite.Ping() != nil {
		http.Error(w, "SQLite DB Not Ready", 503)
		return
	}
	if app.Postgres != nil && app.Postgres.Ping() != nil {
		http.Error(w, "Postgres DB Not Ready", 503)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func (app App) pageIndex(w http.ResponseWriter, r *http.Request) {
	dbType := r.URL.Query().Get("db")
	if dbType == "" {
		dbType = "sqlite"
	}

	items := []models.Item{}
	activeDB := app.SQLite
	if dbType == "postgres"  {
		activeDB = app.Postgres

	}

	if dbType != "postgres" || app.Postgres != nil {
		err := activeDB.Select(&items, "SELECT * FROM items ORDER BY id ASC")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	tmpl, _ := template.ParseFiles("views/index.html")
	data := AppData{Items: items, Source: dbType}

	// If HTMX requested just the table, render the fragment
	if r.Header.Get("HX-Request") != "" {
		tmpl.ExecuteTemplate(w, "data-table", data)
		return
	}

	tmpl.Execute(w, data)
}

func pageTime(w http.ResponseWriter, r *http.Request) {
	ts := time.Now().Format(time.RFC1123)
	fmt.Fprintf(w, `<button hx-get="/time" hx-swap="outerHTML" class="ping-btn">Verified at: %s</button>`, ts)
}
