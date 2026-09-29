package projectprobe

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzJSONManifest exercises the JS/PHP detector parse paths (D-05):
// readManifest → json.Unmarshal → chains. The fuzz body only writes the
// input and calls both detectors; the invariant is no-panic on any input
// (SC3). Parse-success match (D-04): garbage JSON yields ok=false — never
// asserted, never a crash (Pitfall 3). Seed mode runs under plain
// `go test ./project_probe/...` on all CI OSes.
func FuzzJSONManifest(f *testing.F) {
	f.Add([]byte(`{"name":"acme","version":"1.2.3","description":"A pkg"}`))
	f.Add([]byte(`{"name":"vendor/pkg","description":"A composer package"}`)) // composer norm: no version
	f.Add([]byte(`{"name":`))                                                  // truncated
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectJS(dir) // no-panic is the contract; result ignored
		if err := os.WriteFile(filepath.Join(dir, "composer.json"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectPHP(dir)
	})
}

// FuzzTOMLManifest exercises the Python/Rust detector parse paths (D-05):
// readManifest → readTOMLSection (quote-aware) → chains. Presence-match
// detectors (D-09): garbage TOML still claims ok=true — the bool is never
// asserted (Pitfall 3); no-panic is the only invariant.
func FuzzTOMLManifest(f *testing.F) {
	f.Add([]byte("[project]\nname = \"acme\"\nversion = \"1.2.3\"\ndescription = \"A pkg\"\n"))
	f.Add([]byte("[tool.poetry]\nname = \"acme\"\nversion = \"2.0.0\"\n"))
	f.Add([]byte("[project]\nname = \"acme\"\ndynamic = [\"version\"]\n"))
	f.Add([]byte("[package]\nname = \"crate\"\nversion = \"0.1.0\"\ndescription = \"A crate\"\n"))
	f.Add([]byte("[package]\nname = \"crate\"\nversion.workspace = true\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectPython(dir) // no-panic is the contract; result ignored
		if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectRust(dir)
	})
}

// FuzzXMLManifest exercises the C#/.NET and Java/Kotlin detector parse paths
// (D-05): readManifest → xml.Unmarshal → chains. The .csproj fixture is
// always a real project name — a bare ".csproj" file is non-matchable under
// the exact-name guard (13 IN-03). Presence-match detectors: the ok bool is
// never asserted (Pitfall 3); no-panic is the only invariant.
func FuzzXMLManifest(f *testing.F) {
	f.Add([]byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><AssemblyName>MyApp</AssemblyName><Version>1.2.3</Version></PropertyGroup></Project>`))
	f.Add([]byte(`<Project ToolsVersion="15.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003"><PropertyGroup><AssemblyName>Legacy</AssemblyName><Version>2.0.0</Version></PropertyGroup></Project>`))
	f.Add([]byte(`<project><modelVersion>4.0.0</modelVersion><parent><groupId>com.acme</groupId><artifactId>parent</artifactId><version>1.0.0</version></parent><artifactId>child</artifactId><name>Child</name></project>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "MyApp.csproj"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectCSharp(dir) // no-panic is the contract; result ignored
		if err := os.WriteFile(filepath.Join(dir, "pom.xml"), data, 0o600); err != nil {
			t.Skipf("write: %v", err)
		}
		detectJavaKotlin(dir)
	})
}