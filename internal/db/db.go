package db

import (
	"context"

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

func ConnectSQLite(ctx context.Context, dbPath string) (*sqlx.DB, error) {
	// Driver name is "sqlite3" for the CGO driver
	db, err := sqlx.ConnectContext(ctx, "sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	setupDB(ctx, db)
	return db, nil
}

func ConnectPostgres(ctx context.Context, dsn string) (*sqlx.DB, error) {
	// Using pgx stdlib bridge as discussed
	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, err
	}
	setupDB(ctx, db)
	return db, nil
}

func setupDB(ctx context.Context, db *sqlx.DB) {
	switch db.DriverName() {
	case "sqlite3":
		db.MustExecContext(ctx, sqliteCreateTable)
	case "pgx":
		db.MustExecContext(ctx, postgresCreateTable)
	default:
		return
	}

	// Check if we need to seed
	var count int
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM items")
	if err == nil && count == 0 {
		seed(ctx, db)
	}
}

func seed(ctx context.Context, db *sqlx.DB) {
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
		db.MustExecContext(ctx, "INSERT INTO items (title, status) VALUES ($1, $2)", item.Title, item.Status)
	}
}
