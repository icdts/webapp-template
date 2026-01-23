package db

import (
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var sqliteCreateTable = `
	CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
var postgresCreateTable = `
	CREATE TABLE IF NOT EXISTS items (
			id SERIAL PRIMARY KEY, 
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

func ConnectSQLite(dbPath string) (*sqlx.DB, error) {
	// Driver name is "sqlite3" for the CGO driver
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	setupDB(db)
	return db, nil
}

func ConnectPostgres(dsn string) (*sqlx.DB, error) {
	// Using pgx stdlib bridge as discussed
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	setupDB(db)
	return db, nil
}

func setupDB(db *sqlx.DB) {
	switch db.DriverName() {
		case "sqlite3":	
			db.MustExec(sqliteCreateTable)
		case "pgx": 
			db.MustExec(postgresCreateTable)
		default:
			return
	}

	// Check if we need to seed
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM items")
	if err == nil && count == 0 {
		log.Println("Seeding database...")
		seed(db)
	}
}

func seed(db *sqlx.DB) {
	items := []struct {
		Title  string
		Status string
	}{
		{"Static Assets", "Check Red Hat Red header"},
		{"SQLite Persistence", "Restart; check Ref #1"},
		{"HTMX Swapping", "Click 'Ping'; check Network tab"},
		{"CGO Driver", "Check 'readyz' endpoint"},
	}

	for _, item := range items {
		db.MustExec("INSERT INTO items (title, status) VALUES ($1, $2)", item.Title, item.Status)
	}
}
