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
		fmt.Println("postgres not available:", err)
		return
	}
	defer c.Close()

	_ = c.Set(context.Background(), "example", "pg-value")
	val, _ := c.Get(context.Background(), "example")
	fmt.Println(val)

	// Output: pg-value
}
