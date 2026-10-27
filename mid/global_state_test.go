package mid_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGlobalStateTestsDoNotRunInParallel enforces the AGENTS.md rule that tests
// touching mid's package-level collectFuncs slice must not call t.Parallel().
// collectFuncsMu prevents a data race, not one test observing another's swapped
// collector list, so such tests have to run one at a time.
func TestGlobalStateTestsDoNotRunInParallel(t *testing.T) {
	t.Parallel()

	for _, filename := range []string{"machineid_test.go", "machineid_linux_test.go"} {
		assertNoParallelGlobalStateTests(t, filename)
	}
}

func assertNoParallelGlobalStateTests(t *testing.T, filename string) {
	t.Helper()

	source, err := os.ReadFile(filename) //nolint:gosec // reads this package's own test sources
	require.NoError(t, err)

	file, err := parser.ParseFile(token.NewFileSet(), filename, source, 0)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && parallelGlobalStateTest(fn) {
			t.Errorf("%s: %s calls t.Parallel() but reads or writes collectFuncs", filename, fn.Name.Name)
		}
	}
}

// parallelGlobalStateTest reports whether fn calls t.Parallel() and also
// touches collectFuncs, directly or through MachineID.
func parallelGlobalStateTest(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	var parallel, touchesGlobalState bool

	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.SelectorExpr:
			if n.Sel.Name == "Parallel" {
				parallel = true
			}
		case *ast.Ident:
			if n.Name == "collectFuncs" || n.Name == "MachineID" {
				touchesGlobalState = true
			}
		}

		return true
	})

	return parallel && touchesGlobalState
}
