# profile

Loads `<scope>.yml` over `<default>.yml` from a directory and returns the merged result
as YAML.

Part of [`config`](../README.md) in [`github.com/guionardo/go`](../../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/config/profile)

## Install

```bash
go get github.com/guionardo/go/config/profile
```

## Usage

```go
merged, err := profile.GetScopedProfileContent("CONFIGS", "default", "production")
if err != nil {
	log.Fatal(err)
}

var cfg AppConfig
err = yaml.Unmarshal(merged, &cfg)
```

```text
CONFIGS/
  default.yml     # read first
  production.yml  # merged over it
```

Both files are located with the extension omitted or given as `.yml`, `.yaml`, `.YML`,
or `.YAML` — `findYAMLFile` tries `""`, `.yml`, `.yaml`, `.YML`, `.YAML` in that order.
The runnable version is in [`example_test.go`](example_test.go).

## Behaviour

**Both files are mandatory.** `findYAMLFile` returns `file not found: <path>` when
either `defaultScope` or `scope` fails to resolve to a regular file, so
`GetScopedProfileContent` errors rather than falling back to a partial configuration.
`config.Provider` downgrades that error to a warning and carries on with environment
variables alone; called directly, it is an error you must handle. An existing but
*empty* file also fails, as a YAML `EOF` decode error.

**Merging inherits the merger's type rule.** The scope map is merged with
[`merger.MergeMaps`](../merger/README.md), so a scope value whose type differs from the
default's is silently ignored: `port: "9090"` (string) does not replace `port: 8080`
(int), and `9090.5` does not either. Keys present in only one file come through as-is,
and nested maps merge key by key.

**Scope names may not escape the directory.** Both `defaultScope` and `scope` are
checked with `filepath.Rel` against the cleaned base path, so `../evil` — or a sibling
sharing the base name as a prefix, like `../guarded-evil` against `guarded` — is
rejected with `scope "…" escapes base path "…"`. A leading `/` is not an escape: the
scope is rejoined under the base path and fails as `file not found`. Passing the same
name for both scopes reads one file twice, which is how
`config.WithSingleConfigFile` satisfies the two-file contract.

Output is `yaml.Marshal`ed, so keys come back alphabetically ordered with four-space
indentation — fine for unmarshalling, not a byte-for-byte copy of the inputs.
