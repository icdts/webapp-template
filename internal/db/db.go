package db

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Import driver
)

var schema = `
CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func Connect(dbPath string) (*sqlx.DB, error) {
	mustSeed := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		mustSeed = true
	}

	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db.MustExec(schema)

	if mustSeed {
		log.Println("Seeding database...")
		seed(db)
	}

	return db, nil
}

func seed(db *sqlx.DB) {
	items := []struct {
		Title  string
		Status string
	}{
		{"Static Assets", "Check if title is Red Hat Red"},
		{"SQLite Persistence", "Restart container; check if ID #1 exists"},
		{"HTMX Dynamic Swapping", "Click 'Ping' button; check for timestamp"},
		{"CGO SQLite3 Driver", "Verify 'readyz' returns 'ready'"},
		{"Go Templates", "If you can read this table, it works"},
		{"Vendored Dependencies", "Build with disconnected internet"},
	}

	tx := db.MustBegin()
	for _, item := range items {
		tx.MustExec("INSERT INTO items (title, status) VALUES (?, ?)", item.Title, item.Status)
	}
	tx.Commit()
}
