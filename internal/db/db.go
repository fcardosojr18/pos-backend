package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func Connect(ctx context.Context) *pgxpool.Pool {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is empty")
	}
	var err error
	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	if err = pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	return pool
}

func Pool() *pgxpool.Pool { return pool }

func Close() {
	if pool != nil {
		pool.Close()
	}
}
