package environment_test

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/guionardo/go/config/environment"
)

// serverEnv shows the tag contract: env names the variable, default is the
// fallback used when that variable is unset or empty.
type serverEnv struct {
	Host  string `env:"EXAMPLE_SERVER_HOST" default:"localhost"`
	Port  int    `env:"EXAMPLE_SERVER_PORT" default:"8080"`
	Debug bool   `env:"EXAMPLE_SERVER_DEBUG"`
}

// ExampleGetEnv reads a variable that is set, then one that is not. The second
// lookup falls back to the supplied default. Lookups are exact-match on Unix;
// only Windows matches environment variable names case-insensitively.
func ExampleGetEnv() {
	_ = os.Setenv("EXAMPLE_SERVER_HOST", "example.com")

	defer func() { _ = os.Unsetenv("EXAMPLE_SERVER_HOST") }()

	fmt.Println(environment.GetEnv("EXAMPLE_SERVER_HOST"))
	fmt.Println(environment.GetEnv("EXAMPLE_SERVER_UNSET", "fallback"))
	fmt.Println(environment.GetEnv("EXAMPLE_SERVER_UNSET"))

	// Output:
	// example.com
	// fallback
}

// ExampleParseEnvironment fills a struct from env and default tags. A set
// variable wins over its default tag; an unset variable falls back to the
// default, and a field with neither keeps its zero value.
func ExampleParseEnvironment() {
	// ParseEnvironment reports through log/slog, so point the default logger
	// at a discard handler: the example's output is then exactly its prints.
	previousLogger := slog.Default()

	slog.SetDefault(slog.New(slog.DiscardHandler))

	defer slog.SetDefault(previousLogger)

	_ = os.Setenv("EXAMPLE_SERVER_HOST", "example.com")
	_ = os.Setenv("EXAMPLE_SERVER_PORT", "9090")

	defer func() {
		_ = os.Unsetenv("EXAMPLE_SERVER_HOST")
		_ = os.Unsetenv("EXAMPLE_SERVER_PORT")
	}()

	var cfg serverEnv
	if err := environment.ParseEnvironment(&cfg, nil); err != nil {
		panic(err)
	}

	fmt.Printf("%s:%d debug=%t\n", cfg.Host, cfg.Port, cfg.Debug)

	// Output:
	// example.com:9090 debug=false
}
