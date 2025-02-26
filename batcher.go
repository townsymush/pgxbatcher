// Package batcher provides a utility for executing batches of SQL statements with transaction support using the pgx database driver.
//
// Usage:
//
// 1. Import the batcher package:
//
//	import "github.com/townsymush/pgxbatcher"
//
// 2. Create a pgx.Conn object to connect to your PostgreSQL database:
//
//	connString := "postgresql://username:password@localhost:5432/mydb"
//	conn, err := pgx.Connect(context.Background(), connString)
//	if err != nil {
//	    // handle error
//	}
//	defer conn.Close()
//
// 3. Create a new PGXBatcher object:
//
//	batcher := batcher.New(conn, true)
//
//	The second parameter to New() is a boolean flag that specifies whether to execute the batch within a transaction. If set to true, the batch will be executed within a transaction, otherwise each statement will be executed independently.
//
// 4. Add SQL statements to the batch:
//
//	batcher.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Alice", "alice@example.com"})
//	batcher.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Bob", "bob@example.com"})
//
//	You can add as many SQL statements as you need to the batch using the Queue() method. The first argument is the SQL statement, and the second argument is a slice of interface{} values containing the query parameters.
//
// 5. Execute the batch:
//
//	err := batcher.Execute(context.Background())
//	if err != nil {
//	    // handle error
//	}
//
//	The Execute() method sends the batch to the database for execution. If the batch was created with a transaction, the transaction will be committed after all statements have been executed. If any errors occur during execution, they will be returned as a BatcherErrors value.
//
//	If you don't need to use a transaction, you can create the PGXBatcher object with the transactional flag set to false and each statement in the batch will be executed independently.
//
//	Note that you need to import the "github.com/jackc/pgx/v5" packages to use this utility.
package batcher

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type PGXBatcher struct {
	conn          *pgx.Conn
	queries       []string
	batch         *pgx.Batch
	transactional bool
	executed      bool
}

func New(conn *pgx.Conn, transactional bool) *PGXBatcher {
	b := pgx.Batch{}

	if transactional {
		b.Queue("BEGIN")
	}
	return &PGXBatcher{
		conn:          conn,
		batch:         &b,
		transactional: true,
	}
}

func (p *PGXBatcher) Queue(sql string, args []interface{}) {
	p.batch.Queue(sql, args...)
	p.queries = append(p.queries, sql)
}

func (p *PGXBatcher) Execute(ctx context.Context) error {
	if len(p.queries) < 1 {
		return errors.New("no queries to execute")
	}
	if p.executed {
		return errors.New("this batch has already been executed. Create a new instance or call Reset()")
	}
	if p.transactional {
		p.batch.Queue("COMMIT")
	}
	results := p.conn.SendBatch(ctx, p.batch)
	defer results.Close()
	p.executed = true
	var errs StatementErrors

	for _, q := range p.queries {
		_, err := results.Exec()
		if err != nil {
			errs = append(errs, StatementError{
				err: err,
				sql: q,
			})
		}
	}

	if errs.isErrors() {
		return errs
	}
	return nil
}

func (p *PGXBatcher) Reset() {
	p.batch = &pgx.Batch{}
	p.queries = []string{}
}
