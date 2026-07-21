package release

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const swapperPerms = 0o755

//go:embed swapper/swapper_linux_amd64
//go:embed swapper/swapper_darwin_amd64
//go:embed swapper/swapper_darwin_arm64
//go:embed swapper/swapper_windows_amd64.exe
var swapperBinary embed.FS

func runtimeExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}

func swapperFilename() string {
	return "swapper_" + runtime.GOOS + "_" + runtime.GOARCH + runtimeExeSuffix()
}

func ExtractSwapper(targetDir string) (string, error) {
	name := swapperFilename()

	data, err := swapperBinary.ReadFile("swapper/" + name)
	if err != nil {
		return "", fmt.Errorf("read embedded swapper %s: %w", name, err)
	}

	outPath := filepath.Join(targetDir, name)

	if err := os.WriteFile(outPath, data, swapperPerms); err != nil {
		return "", fmt.Errorf("write swapper: %w", err)
	}

	return outPath, nil
}
