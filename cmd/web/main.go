package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/icdts/webapp"
	"github.com/icdts/webapp/internal/db"
	"github.com/icdts/webapp/internal/models"

	"github.com/jmoiron/sqlx"
)

type App struct {
	SQLite   *sqlx.DB
	Postgres *sqlx.DB
	Log      *slog.Logger
}

type AppData struct {
	Items  []models.Item
	Source string
}

type HandleFuncWithDb func(*sqlx.DB, http.ResponseWriter, *http.Request)
type responseWriterWrapper struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func (app *App) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        wrapped := &responseWriterWrapper{
            ResponseWriter: w,
            statusCode:     http.StatusOK,
        }

        next.ServeHTTP(wrapped, r)

        duration := time.Since(start)

        app.Log.Info("request_completed",
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrapped.statusCode,
            "duration", duration,
            "ip", r.RemoteAddr,
            "user_agent", r.UserAgent(),
        )
    })
}



func main() {

	var err error
	var port int

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	portStr := os.Getenv("PORT")
	if port, err = strconv.Atoi(portStr); err != nil {
		logger.Error("config", "status", "invalid", "param", "PORT", "value", portStr, "error", err)
		os.Exit(1)
	}

	htmxPath := os.Getenv("HTMX_SRC")
	if htmxPath == "" {
		logger.Error("config", "status", "invalid", "param", "HTMX_SRC", "value", htmxPath, "error", "missing")
		os.Exit(1)
	}
	if _, err := os.Stat(htmxPath); os.IsNotExist(err) {
		logger.Error("config", "status", "invalid", "param", "HTMX_SRC", "value", htmxPath, "error", err)
		os.Exit(1)
	}
	logger.Info("config", "status", "valid", "param", "HTMX_SRC", "value", htmxPath)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCancel()
	sqDB, err := db.ConnectSQLite(dbCtx,"tmp/app.db")
	if err != nil {
		logger.Error("db_init", "status", "failed",  "db_type", "sqlite", "path", "tmp/app.db", "error", err)
		os.Exit(1)
	}
	defer sqDB.Close()

	pgDSN := os.Getenv("DATABASE_URL")
	pgDB, err := db.ConnectPostgres(dbCtx,pgDSN)
	if err != nil {
		logger.Warn("db_init", "status", "failed", "db_type", "postgres", "error", err)
	} else {
		defer pgDB.Close()
	}

	app := &App{
		SQLite:   sqDB,
		Postgres: pgDB,
		Log:      logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/readyz", app.readyz)
	mux.HandleFunc("/assets/htmx.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, htmxPath) })
	mux.Handle("/static/", http.FileServer(http.FS(webapp.EmbeddedStatic)))

	mux.HandleFunc("/", app.pageIndex)
	mux.HandleFunc("/time", pageTime)

	handler := app.loggingMiddleware(mux)

	portStr = fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr: portStr,
		Handler: handler,
	}

	go func() {
		logger.Info("server_startup", "address", port)
		// ErrServerClosed is a normal error returned when we call Shutdown(), so we ignore it
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_startup", "address", portStr, "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("server_shutdown", "status", "started")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown", "status", "forced", "error", err)
	}

	logger.Info("server_shutdown", "status", "complete")
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
	if dbType == "postgres" {
		activeDB = app.Postgres

	}

	if dbType != "postgres" || app.Postgres != nil {
		err := activeDB.SelectContext(r.Context(), &items, "SELECT * FROM items ORDER BY id ASC")
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
