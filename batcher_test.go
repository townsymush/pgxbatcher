package pgxbatcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

var conn *pgx.Conn

func TestQueue(t *testing.T) {
	b := New(conn, true)

	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Bob", "bob@example.com")

	if err := b.Execute(context.Background()); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var count int

	if err := conn.QueryRow(context.TODO(), "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Errorf("Failed to query test table: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows in test table, got %d", count)
	}
}

func TestPGXBatcher_Execute_Errors(t *testing.T) {
	b := New(conn, false)

	// add invalid SQL statement to batch
	b.Queue("INVALID SQL", []interface{}{})

	// execute batch
	err := b.Execute(context.TODO())

	// assert error type and message
	if err == nil {
		t.Error("Expected error, but got nil")
	}

	if errs, ok := err.(StatementErrors); ok {
		if len(errs) != 1 {
			t.Errorf("Expected 1 error, but got %d", len(errs))
		}

		/*if errs[0].sql != "INVALID SQL" {
			t.Errorf("Expected error SQL to be 'INVALID SQL', but got '%s'", errs[0].sql)
		}*/
	} else {
		t.Errorf("Expected error of type 'BatcherErrors', but got '%T'", err)
	}
}

func TestPGXBatcher_Execute_Transactional_Errors(t *testing.T) {
	b := New(conn, true)

	// add invalid SQL statement to batch
	b.Queue("INVALID SQL")

	// execute batch
	err := b.Execute(context.TODO())

	// assert error type and message
	if err == nil {
		t.Error("Expected error, but got nil")
	}

	if errs, ok := err.(StatementErrors); ok {
		if len(errs) != 3 {
			t.Errorf("Expected 3 error, but got %d: \n%v", len(errs), errs.Error())
		}

		/*if errs[0].sql != "INVALID SQL" {
			t.Errorf("Expected error SQL to be 'INVALID SQL', but got '%s'", errs[0].sql)
		}*/
	} else {
		t.Errorf("Expected error of type 'BatcherErrors', but got '%T'", err)
	}
}

func TestPGXBatcher_Reset(t *testing.T) {
	b := New(conn, false)

	// Queue some queries
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Alice", "alice@example.com"})
	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Bob", "bob@example.com"})

	// Reset the batcher
	b.Reset()

	// Ensure that the batch is empty after reset
	if b.batch.Len() != 0 {
		t.Errorf("Expected empty batch after Reset(), got %+v", b.batch.Len())
		t.Fail()
	}

	// execute batch
	err := b.Execute(context.TODO())
	if err == nil {
		t.Error("expected batcher will fail with no queries")
		t.Fail()
	}

	if !errors.Is(err, ErrEmptyBatch) {
		t.Errorf("unexpected error %s", err)
	}
}

func TestPGXBatcher_ExecuteExecuted(t *testing.T) {
	b := New(conn, true)

	b.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")

	if err := b.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := b.Execute(context.Background())
	if err == nil {
		t.Fatal("expected an error, but got none")
	}

	if !errors.Is(err, ErrExecutedBatch) {
		t.Errorf("unexpected error message: got %q, want %q", err.Error(), ErrExecutedBatch.Error())
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
	teardown(ctx, conn)
	os.Exit(code)
}
