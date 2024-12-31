package postgres

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	DB *sql.DB
}

func Init(dsn string) *DB {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	return &DB{
		DB: db,
	}
}
func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}
