package config

import (
	"log/slog"
	"maps"
	"reflect"
	"strings"
	"sync"
)

var (
	logger     *slog.Logger
	loggerOnce sync.Once
)

// log returns a new logger for this package
func log() *slog.Logger {
	loggerOnce.Do(func() {
		logger = slog.With(slog.String("module", "config"))
	})

	return logger
}

func getConfigurationLog(c any) slog.Attr {
	configs := getMapFromStruct(c, "")

	attrs := make([]any, 0, len(configs))

	for k, v := range configs {
		attrs = append(attrs, slog.Attr{Key: k, Value: slog.AnyValue(v)})
	}

	return slog.Group(reflect.TypeOf(c).Name(), attrs...)
}

// getMapFromStruct returns a map representation, removing the secrets fields for logging
func getMapFromStruct(c any, parentPath string) map[string]any {
	if reflect.TypeOf(c).Kind() != reflect.Struct {
		return map[string]any{
			parentPath: c,
		}
	}

	attrs := map[string]any{}

	for i := 0; i < reflect.TypeOf(c).NumField(); i++ {
		field := reflect.TypeOf(c).Field(i)

		if _, safe := field.Tag.Lookup("safe"); safe {
			attrs[fieldPath(parentPath, field.Name)] = "********"
			continue
		}

		if field.Type.Kind() == reflect.Struct {
			subStruct := getMapFromStruct(reflect.ValueOf(c).Field(i).Interface(), fieldPath(parentPath, field.Name))
			maps.Copy(attrs, subStruct)

			continue
		}

		attrs[fieldPath(parentPath, field.Name)] = reflect.ValueOf(c).Field(i).Interface()
	}

	return attrs
}
func fieldPath(fields ...string) string {
	return strings.TrimPrefix(strings.Join(fields, "."), ".")
}
