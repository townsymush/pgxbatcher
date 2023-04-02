package batcher

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	pool *pgxpool.Pool
)

func setup() error {
	var err error

	username := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	db := os.Getenv("POSTGRES_DB")
	host := os.Getenv("POSTGRES_HOST")

	url := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", username, password, host, db)
	pool, err = pgxpool.Connect(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	_, err = pool.Exec(context.Background(), "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		return fmt.Errorf("failed to create test table: %v", err)
	}
	return nil
}

func teardown() error {
	_, err := pool.Exec(context.Background(), "DROP TABLE users")
	if err != nil {
		return fmt.Errorf("failed to drop test table: %v", err)
	}
	pool.Close()
	return nil
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up tests: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := teardown(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to tear down tests: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func TestQueue(t *testing.T) {
	batcher := New(*pool, true)
	batcher.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Alice", "alice@example.com"})
	batcher.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Bob", "bob@example.com"})
	err := batcher.Execute(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query test table: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows in test table, got %d", count)
	}
}
