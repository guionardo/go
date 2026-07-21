package environment

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
)

// GetEnv returns the value of the environment variable, or a default if not set.
func GetEnv(env string, defaultValue ...string) string {
	if env == "" {
		return ""
	}

	if value, ok := os.LookupEnv(env); ok {
		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

// ParseEnvironment parses the environment variables into a struct
// It returns an error if the environment variables are invalid
// The argument must be a pointer to a struct
func ParseEnvironment(s any, parentType reflect.Type) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			slog.Error("panic in ParseEnvironment", "panic", panicErr, "stack", string(debug.Stack()))
			err = fmt.Errorf("panic: %v", panicErr)
		}
	}()

	t := reflect.TypeOf(s)

	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", t.Kind())
	}

	var (
		envName     string
		envValue    string
		envFound    bool
		setEnvs     = map[string]string{} // field:env name
		missingEnvs = map[string]string{} // field:env name
	)

	for i := 0; i < t.Elem().NumField(); i++ {
		field := t.Elem().Field(i)

		if field.Type.Kind() == reflect.Struct {
			fieldValue := reflect.ValueOf(s).Elem().Field(i)
			if parseErr := ParseEnvironment(fieldValue.Addr().Interface(), t); parseErr != nil {
				err = errors.Join(err, parseErr)
			}

			continue
		}

		envValue, envName, envFound = getFieldEnvValue(field)
		if !envFound {
			missingEnvs[field.Name] = envName
		}

		if envValue == "" {
			continue
		}

		fieldValue := reflect.ValueOf(s).Elem().Field(i)

		if fieldValue.CanSet() {
			if setErr := setField(field, fieldValue, envValue); setErr != nil {
				err = errors.Join(err, setErr)
			} else {
				setEnvs[field.Name] = envName
			}
		}
	}
	// debug log
	if len(setEnvs) > 0 {
		logArgs := []any{slog.String("instance", t.String())}
		for fieldName, envName := range setEnvs {
			logArgs = append(logArgs, slog.String(fieldName, envName))
		}

		slog.Debug("config.SetEnv", logArgs...)
	}

	if len(missingEnvs) > 0 {
		logArgs := []any{slog.String("instance", t.String())}
		for fieldName, envName := range missingEnvs {
			logArgs = append(logArgs, slog.String(fieldName, envName))
		}

		slog.Debug("config.SetEnv MISSING ENVS", logArgs...)
	}

	return err
}

// getFieldEnvValue returns the environment value for a field or the default value if not set
func getFieldEnvValue(field reflect.StructField) (value string, envName string, found bool) {
	if envName = field.Tag.Get("env"); envName != "" {
		if envValue := os.Getenv(envName); envValue != "" {
			return envValue, envName, true
		}
	}

	return field.Tag.Get("default"), envName, false
}

func setField(field reflect.StructField, fieldValue reflect.Value, envValue string) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			slog.Error("panic in setField", "field", field.Name, "panic", panicErr, "stack", string(debug.Stack()))
			err = fmt.Errorf("panic setting field %s: %v", field.Name, panicErr)
		}
	}()

	switch field.Type.Kind() {
	case reflect.String:
		fieldValue.SetString(envValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intValue int64

		if intValue, err = strconv.ParseInt(envValue, 10, 64); err == nil {
			fieldValue.SetInt(intValue)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintValue uint64

		if uintValue, err = strconv.ParseUint(envValue, 10, 64); err == nil {
			fieldValue.SetUint(uintValue)
		}

	case reflect.Bool:
		var boolValue bool

		if boolValue, err = strconv.ParseBool(envValue); err == nil {
			fieldValue.SetBool(boolValue)
		}

	case reflect.Float64, reflect.Float32:
		var floatValue float64

		if floatValue, err = strconv.ParseFloat(envValue, 64); err == nil {
			fieldValue.SetFloat(floatValue)
		}

	case reflect.Struct:
		if err = ParseEnvironment(fieldValue.Addr().Interface(), nil); err != nil {
			return fmt.Errorf("invalid struct value for field %s: %w", field.Name, err)
		}
	}

	if err != nil {
		err = fmt.Errorf("invalid field value '%s' (%s) for field %s: %w", envValue,
			field.Type.Kind().String(), field.Name, err)
	}

	return err
}
