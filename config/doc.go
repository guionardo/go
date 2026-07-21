// Package config provides a generic typed configuration provider.
//
// It loads YAML profiles by scope (e.g., "production", "development"),
// overlays environment variable overrides via struct tags, and validates
// the result using go-playground/validator.
//
// Key types:
//   - Provider[T]: thread-safe configuration provider
//   - Logger: minimal logging interface (can wrap slog, log, zap, etc.)
//
// Provider[T]:
//
//	p := config.NewProvider[AppConfig]()
//	cfg, err := p.GetConfiguration()
//	err = p.UpdateConfiguration(cfg)
//
// Options:
//   - WithProfilesPath: set YAML profile directory
//   - WithScope: set active scope name
//   - WithDefaultScope: set fallback scope
//   - WithLogger: inject custom logger
//   - WithDebugLogger: enable debug logging
//
// Sub-packages:
//   - config/environment: env-var parsing via struct tags
//   - config/profile: YAML profile loading and merging
//   - config/merger: recursive deep-merge of map[string]any
//   - config/validation: struct validation via Validator interface
package config
