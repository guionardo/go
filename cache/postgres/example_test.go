//go:build e2e

package postgres_test

import (
	"context"
	"fmt"
	"os"

	"github.com/guionardo/go/cache/postgres"
)

func ExampleNew() {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://localhost:5432/cache_test?sslmode=disable"
	}

	c, err := postgres.New[string, string](postgres.WithConnString(connString))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if err := c.Set(context.Background(), "example", "pg-value"); err != nil {
		fmt.Println("error:", err)
		return
	}

	value, err := c.Get(context.Background(), "example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(value)

	if err := c.Close(); err != nil {
		fmt.Println("error:", err)
	}

	// Output: pg-value
}
