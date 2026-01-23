package models

import "time"

type Item struct {
	ID        int       `db:"id"`
	Title     string    `db:"title"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}
