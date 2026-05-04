// Package envloader populates structs from environment variables using struct tags.
//
// Fields are mapped via the `env` struct tag. Supported tag options:
//
//	`env:"PORT"`               – required field
//	`env:"PORT,optional"`      – silently skipped if the env var is not set
//	`env:"PORT,default=8080"`  – uses the given value if the env var is not set
//
// Supported field types: string, bool, int/int8/int16/int32/int64,
// uint/uint8/uint16/uint32/uint64, and pointers to any of the above.
// Pointer fields are set to nil when the env var is absent and no default is given.
// Nested structs are traversed recursively. Unexported fields are ignored.
//
// Example:
//
//	type Config struct {
//	    AppName string `env:"APP_NAME"`
//	    Port    int    `env:"PORT,default=8080"`
//	    Debug   bool   `env:"DEBUG,optional"`
//	}
//
//	var cfg Config
//	if err := envloader.Load(&cfg); err != nil {
//	    log.Fatal(err)
//	}
package envloader

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Load populates the fields of the struct pointed to by data from environment
// variables. data must be a non-nil pointer to a struct.
func Load(data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Pointer || val.IsNil() || val.Elem().Kind() != reflect.Struct {
		return &DataTypeError{Type: reflect.TypeOf(data)}
	}
	return populateStructFields(val.Elem())
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

// DataTypeError is returned when Load is called with a value that is not a
// non-nil pointer to a struct.
type DataTypeError struct {
	Type reflect.Type
}

func (e *DataTypeError) Error() string {
	if e.Type == nil {
		return "load: Load(nil)"
	}
	if e.Type.Kind() != reflect.Pointer {
		return "load: Load(non-pointer " + e.Type.String() + ")"
	}
	if e.Type.Elem().Kind() != reflect.Struct {
		return "load: Load(non-struct pointer " + e.Type.String() + ")"
	}
	return "load: Load(nil " + e.Type.String() + ")"
}

// EnvTagMissingError is returned when an exported struct field has no `env` tag.
type EnvTagMissingError struct {
	Field        reflect.StructField
	ParentStruct reflect.Type
}

func (e *EnvTagMissingError) Error() string {
	return fmt.Sprintf("load: missing env tag for field %s.%s", e.ParentStruct.Name(), e.Field.Name)
}

// EnvValueMissingError is returned when a required env var is not set.
type EnvValueMissingError struct {
	Field        reflect.StructField
	ParentStruct reflect.Type
	EnvVar       string
}

func (e *EnvValueMissingError) Error() string {
	return fmt.Sprintf("load: env var %q not set for field %s.%s",
		e.EnvVar, e.ParentStruct.Name(), e.Field.Name)
}

// EnvValueParseError is returned when an env var value cannot be parsed into
// the target field type. Unwrap returns the underlying strconv error.
type EnvValueParseError struct {
	Field        reflect.StructField
	ParentStruct reflect.Type
	EnvVar       string
	Value        string
	Err          error
}

func (e *EnvValueParseError) Error() string {
	return fmt.Sprintf("load: failed to parse env var %s=%q for field %s.%s: %v",
		e.EnvVar, e.Value, e.ParentStruct.Name(), e.Field.Name, e.Err)
}

func (e *EnvValueParseError) Unwrap() error { return e.Err }

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// tagOptions holds the parsed contents of an `env` struct tag.
type tagOptions struct {
	name         string
	optional     bool
	hasDefault   bool
	defaultValue string
}

// parseTag splits a raw tag value like "PORT,default=8080" into its parts.
func parseTag(raw string) tagOptions {
	name, rest, _ := strings.Cut(raw, ",")
	opts := tagOptions{name: name}
	for _, opt := range strings.Split(rest, ",") {
		opt = strings.TrimSpace(opt)
		switch {
		case opt == "optional":
			opts.optional = true
		case strings.HasPrefix(opt, "default="):
			opts.hasDefault = true
			opts.defaultValue = strings.TrimPrefix(opt, "default=")
		}
	}
	return opts
}

// populateStructFields iterates over struct fields and populates them from
// environment variables.
func populateStructFields(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		if !field.CanSet() {
			continue // skip unexported fields
		}

		// Recurse into embedded/nested structs.
		if field.Kind() == reflect.Struct {
			if err := populateStructFields(field); err != nil {
				return err
			}
			continue
		}

		// Dereference a single pointer level so the rest of the logic is uniform.
		isPointer := field.Kind() == reflect.Pointer
		targetKind := field.Kind()
		if isPointer {
			targetKind = field.Type().Elem().Kind()
		}

		rawTag := structField.Tag.Get("env")
		if rawTag == "" {
			return &EnvTagMissingError{ParentStruct: typ, Field: structField}
		}

		opts := parseTag(rawTag)

		envValue, exists := os.LookupEnv(opts.name)
		switch {
		case exists:
			// use the env var value — handled below
		case opts.hasDefault:
			envValue = opts.defaultValue
		case opts.optional || isPointer:
			// Leave pointer fields as nil, zero-value fields as-is.
			continue
		default:
			return &EnvValueMissingError{Field: structField, ParentStruct: typ, EnvVar: opts.name}
		}

		parsed, err := parseValue(targetKind, envValue)
		if err != nil {
			return &EnvValueParseError{
				Field:        structField,
				ParentStruct: typ,
				EnvVar:       opts.name,
				Value:        envValue,
				Err:          err,
			}
		}

		if isPointer {
			ptr := reflect.New(field.Type().Elem())
			ptr.Elem().Set(reflect.ValueOf(parsed).Convert(field.Type().Elem()))
			field.Set(ptr)
		} else {
			field.Set(reflect.ValueOf(parsed).Convert(field.Type()))
		}
	}
	return nil
}

// parseValue converts the string s into a Go value matching kind.
func parseValue(kind reflect.Kind, s string) (any, error) {
	switch kind {
	case reflect.String:
		return s, nil

	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, err
		}
		return b, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return n, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return n, nil

	default:
		return nil, fmt.Errorf("unsupported field type: %s", kind)
	}
}
