package pgxbatcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var conn *pgx.Conn

func TestQueue(t *testing.T) {
	b := New(conn, true)
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Bob", "bob@example.com")
	err := b.Execute(context.TODO())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var count int
	err = conn.QueryRow(context.TODO(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query test table: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows in test table, got %d", count)
	}
}

func TestPGXBatcher_Execute_Errors(t *testing.T) {
	b := New(conn, false)

	// add invalid SQL statement to batch
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
	b.Queue("INVALID SQL")

	// execute batch
	err := b.Execute(context.TODO())

	// assert error type and message
	if err == nil {
		t.Error("Expected error, but got nil")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "42601" { // Syntax error
			t.Errorf("Expected syntax error with code 42601, got %s", pgErr.Code)
			t.FailNow()
		}
		return
	}

	t.Error("expected err of type *pgconn.PgError, but got none")
	t.FailNow()
}

func TestPGXBatcher_Reset(t *testing.T) {
	b := New(conn, false)

	// Queue some queries
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Bob", "bob@example.com")

	// Reset the b
	b.Reset()

	// Ensure that the batch is empty after reset
	if len(b.queries) != 0 {
		t.Errorf("Expected empty batch after Reset(), got %+v", b.queries)
		t.Fail()
	}

	if b.batch.Len() != 0 {
		t.Errorf("Expected empty batch after Reset(), got %+v", b.batch)
		t.FailNow()
	}

	// execute batch
	err := b.Execute(context.TODO())
	if !errors.Is(err, ErrEmptyBatch) {
		t.Errorf("expected an error of type ErrEmptyBatch")
		t.FailNow()
	}
}

func TestPGXBatcher_ExecuteExecuted(t *testing.T) {
	b := New(conn, true)

	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")

	err := b.Execute(context.TODO())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = b.Execute(context.TODO())
	if err == nil {
		t.Error("expected an error, but got none")
		t.FailNow()
	}

	if !errors.Is(err, ErrExecutedBatch) {
		t.Errorf("expected an error of type ErrExecutedBatch")
		t.FailNow()
	}
}

func teardown(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "DROP TABLE IF EXISTS users")
	if err != nil {
		return fmt.Errorf("failed to drop test table: %v", err)
	}
	conn.Close(ctx)
	return nil
}

func setupDBConnection(ctx context.Context) (*pgx.Conn, error) {
	username := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	db := os.Getenv("POSTGRES_DB")
	host := os.Getenv("POSTGRES_HOST")

	url := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", username, password, host, db)
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return conn, fmt.Errorf("failed to connect to database: %v", err)
	}
	_, err = conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		return conn, fmt.Errorf("failed to create test table: %v", err)
	}
	return conn, nil
}

func TestMain(m *testing.M) {
	var err error
	ctx := context.Background()
	conn, err = setupDBConnection(ctx)
	if err != nil {
		fmt.Printf("could not set up database connection: %s", err)
		os.Exit(1)
	}
	code := m.Run()
	if err = teardown(ctx, conn); err != nil {
		fmt.Printf("could not tear down tests: %s", err)
	}
	os.Exit(code)
}
