package merger_test

import (
	"fmt"

	"github.com/guionardo/go/config/merger"
)

func ExampleMergeMaps() {
	base := map[string]any{
		"host": "localhost",
		"port": 8080,
		"server": map[string]any{
			"timeout": 30,
			"tls":     false,
		},
	}
	production := map[string]any{
		"port": 9090,
		"server": map[string]any{
			"tls": true,
		},
	}

	merged := merger.MergeMaps(base, production)

	server := merged["server"].(map[string]any)
	fmt.Println(merged["host"], merged["port"])
	fmt.Printf("timeout=%v tls=%v\n", server["timeout"], server["tls"])
	// Output:
	// localhost 9090
	// timeout=30 tls=true
}
