// Package profile provides YAML profile loading and merging for
// configuration scopes.
//
// Loads YAML files from a base directory path. Merges default scope
// with the active scope. Supports .yml, .yaml, .YML, .YAML extensions.
// Includes path traversal protection.
//
// Usage:
//
//	data, err := profile.GetScopedProfileContent("/etc/app", "default", "production")
package profile
