# PGX Batcher

PGX Wrapper is a simple Go utility that provides a wrapper around `pgx.Batch`, allowing you to execute a batch of SQL statements with transaction support in a single network round-trip using the [Jackc PGX](https://github.com/jackc/pgx) database driver. This package is designed to simplify the process of executing multiple SQL statements in a batch, while providing transaction support and error handling

## Installation

To install the package, run the following command:

```go
go get https://github.com/townsymush/pgxbatcher
```

## Usage

Here's an example of how to use the pakage to execute a batch of SQL statements:

```go
package main

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v4/pgxpool"
    "github.com/townsymush/pgxbatcher"
)

func main() {
    // Create a pgxpool.Pool object to connect to your PostgreSQL database
    connString := "postgresql://username:password@localhost:5432/mydb"
    pool, err := pgxpool.Connect(context.Background(), connString)
    if err != nil {
        // handle error
    }
    defer pool.Close()

    // Create a new PGXBatcher object
    batcher := pgxbatcher.New(pool, true)

    // Add SQL statements to the batch
    batcher.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Alice", "alice@example.com"})
    batcher.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", []interface{}{"Bob", "bob@example.com"})

    // Execute the batch
    err = batcher.Execute(context.Background())
    if err != nil {
        // handle errors. Note the Error type is StatementErrors []StatementError which will return all errors a string with the sql statement if required
    }

    fmt.Println("Batch executed successfully!")
}
```

# Contributing
If you find a bug or have a feature request, please open an issue on the GitHub repository. Pull requests are also welcome!

# License

This package is licensed under the MIT License.