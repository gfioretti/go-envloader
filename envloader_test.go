package envloader

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// setenv sets key=value for the duration of a test and restores the previous
// state via t.Cleanup, so tests cannot leak env vars into each other.
func setenv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

// ---------------------------------------------------------------------------
// Load – argument validation
// ---------------------------------------------------------------------------

func TestLoad_NilArgument(t *testing.T) {
	err := Load(nil)

	if _, ok := errors.AsType[*DataTypeError](err); !ok {
		t.Fatalf("expected DataTypeError, got %T: %v", err, err)
	}
}

func TestLoad_NonPointer(t *testing.T) {
	type Config struct {
		Name string `env:"NAME"`
	}

	err := Load(Config{})

	if _, ok := errors.AsType[*DataTypeError](err); !ok {
		t.Fatalf("expected DataTypeError, got %T: %v", err, err)
	}
}

func TestLoad_PointerToNonStruct(t *testing.T) {
	err := Load(new("hello"))

	if _, ok := errors.AsType[*DataTypeError](err); !ok {
		t.Fatalf("expected DataTypeError, got %T: %v", err, err)
	}
}

func TestLoad_NilPointer(t *testing.T) {
	type Config struct{}
	var cfg *Config

	err := Load(cfg)

	if _, ok := errors.AsType[*DataTypeError](err); !ok {
		t.Fatalf("expected DataTypeError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// Load – scalar types
// ---------------------------------------------------------------------------

func TestLoad_StringField(t *testing.T) {
	type Config struct {
		Name string `env:"TEST_NAME"`
	}
	setenv(t, "TEST_NAME", "gopher")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "gopher" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "gopher")
	}
}

func TestLoad_IntField(t *testing.T) {
	type Config struct {
		Port int `env:"TEST_PORT"`
	}
	setenv(t, "TEST_PORT", "9090")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port: got %d, want 9090", cfg.Port)
	}
}

func TestLoad_Int64Field(t *testing.T) {
	type Config struct {
		Timeout int64 `env:"TEST_TIMEOUT"`
	}
	setenv(t, "TEST_TIMEOUT", "3600")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timeout != 3600 {
		t.Errorf("Timeout: got %d, want 3600", cfg.Timeout)
	}
}

func TestLoad_UintField(t *testing.T) {
	type Config struct {
		Workers uint `env:"TEST_WORKERS"`
	}
	setenv(t, "TEST_WORKERS", "4")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers: got %d, want 4", cfg.Workers)
	}
}

func TestLoad_BoolField_True(t *testing.T) {
	type Config struct {
		Debug bool `env:"TEST_DEBUG"`
	}

	for _, raw := range []string{"true", "TRUE", "1", "True"} {
		t.Run(raw, func(t *testing.T) {
			setenv(t, "TEST_DEBUG", raw)

			var cfg Config
			if err := Load(&cfg); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !cfg.Debug {
				t.Errorf("Debug: got false, want true for raw value %q", raw)
			}
		})
	}
}

func TestLoad_BoolField_False(t *testing.T) {
	type Config struct {
		Debug bool `env:"TEST_DEBUG"`
	}

	for _, raw := range []string{"false", "FALSE", "0"} {
		t.Run(raw, func(t *testing.T) {
			setenv(t, "TEST_DEBUG", raw)

			var cfg Config
			if err := Load(&cfg); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Debug {
				t.Errorf("Debug: got true, want false for raw value %q", raw)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Load – pointer fields
// ---------------------------------------------------------------------------

func TestLoad_PointerField_Set(t *testing.T) {
	type Config struct {
		Port *int `env:"TEST_PORT"`
	}
	setenv(t, "TEST_PORT", "8080")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port == nil {
		t.Fatal("Port: got nil, want non-nil pointer")
	}
	if *cfg.Port != 8080 {
		t.Errorf("*Port: got %d, want 8080", *cfg.Port)
	}
}

func TestLoad_PointerField_AbsentIsNil(t *testing.T) {
	type Config struct {
		Port *int `env:"TEST_PORT_ABSENT"`
	}

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != nil {
		t.Errorf("Port: got %v, want nil", cfg.Port)
	}
}

func TestLoad_PointerStringField(t *testing.T) {
	type Config struct {
		Region *string `env:"TEST_REGION"`
	}
	setenv(t, "TEST_REGION", "eu-west-1")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Region == nil || *cfg.Region != "eu-west-1" {
		t.Errorf("Region: got %v, want %q", cfg.Region, "eu-west-1")
	}
}

// ---------------------------------------------------------------------------
// Load – tag options
// ---------------------------------------------------------------------------

func TestLoad_OptionalField_Present(t *testing.T) {
	type Config struct {
		LogLevel string `env:"TEST_LOG_LEVEL,optional"`
	}
	setenv(t, "TEST_LOG_LEVEL", "debug")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoad_OptionalField_Absent(t *testing.T) {
	type Config struct {
		LogLevel string `env:"TEST_LOG_LEVEL_ABSENT,optional"`
	}

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "" {
		t.Errorf("LogLevel: got %q, want empty string", cfg.LogLevel)
	}
}

func TestLoad_DefaultValue_EnvAbsent(t *testing.T) {
	type Config struct {
		Port int `env:"TEST_PORT_DEFAULT,default=3000"`
	}

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("Port: got %d, want 3000", cfg.Port)
	}
}

func TestLoad_DefaultValue_EnvPresent(t *testing.T) {
	// Env var should win over the default.
	type Config struct {
		Port int `env:"TEST_PORT_DEFAULT,default=3000"`
	}
	setenv(t, "TEST_PORT_DEFAULT", "9000")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port: got %d, want 9000", cfg.Port)
	}
}

func TestLoad_DefaultString(t *testing.T) {
	type Config struct {
		Env string `env:"TEST_ENV_NAME,default=production"`
	}

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != "production" {
		t.Errorf("Env: got %q, want %q", cfg.Env, "production")
	}
}

// ---------------------------------------------------------------------------
// Load – nested structs
// ---------------------------------------------------------------------------

func TestLoad_NestedStruct(t *testing.T) {
	type DBConfig struct {
		Host string `env:"TEST_DB_HOST"`
		Port int    `env:"TEST_DB_PORT"`
	}
	type Config struct {
		AppName string `env:"TEST_APP_NAME"`
		DB      DBConfig
	}

	setenv(t, "TEST_APP_NAME", "my-service")
	setenv(t, "TEST_DB_HOST", "localhost")
	setenv(t, "TEST_DB_PORT", "5432")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppName != "my-service" {
		t.Errorf("AppName: got %q, want %q", cfg.AppName, "my-service")
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host: got %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port: got %d, want 5432", cfg.DB.Port)
	}
}

// ---------------------------------------------------------------------------
// Load – unexported fields are skipped
// ---------------------------------------------------------------------------

func TestLoad_UnexportedFieldsSkipped(t *testing.T) {
	type Config struct {
		Name    string `env:"TEST_NAME"`
		private string // no tag — should be silently skipped
	}
	setenv(t, "TEST_NAME", "visible")

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "visible" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "visible")
	}
}

// ---------------------------------------------------------------------------
// Load – error cases
// ---------------------------------------------------------------------------

func TestLoad_MissingEnvTag(t *testing.T) {
	type Config struct {
		Name string // no env tag
	}

	var cfg Config
	err := Load(&cfg)

	target, ok := errors.AsType[*EnvTagMissingError](err)
	if !ok {
		t.Fatalf("expected EnvTagMissingError, got %T: %v", err, err)
	}
	if target.Field.Name != "Name" {
		t.Errorf("Field.Name: got %q, want %q", target.Field.Name, "Name")
	}
}

func TestLoad_MissingRequiredEnvVar(t *testing.T) {
	type Config struct {
		Secret string `env:"TEST_SECRET_ABSENT"`
	}

	var cfg Config
	err := Load(&cfg)

	target, ok := errors.AsType[*EnvValueMissingError](err)
	if !ok {
		t.Fatalf("expected EnvValueMissingError, got %T: %v", err, err)
	}
	if target.EnvVar != "TEST_SECRET_ABSENT" {
		t.Errorf("EnvVar: got %q, want %q", target.EnvVar, "TEST_SECRET_ABSENT")
	}
}

func TestLoad_InvalidIntValue(t *testing.T) {
	type Config struct {
		Port int `env:"TEST_PORT_BAD"`
	}
	setenv(t, "TEST_PORT_BAD", "not-a-number")

	var cfg Config
	err := Load(&cfg)

	target, ok := errors.AsType[*EnvValueParseError](err)
	if !ok {
		t.Fatalf("expected EnvValueParseError, got %T: %v", err, err)
	}
	if target.EnvVar != "TEST_PORT_BAD" {
		t.Errorf("EnvVar: got %q, want %q", target.EnvVar, "TEST_PORT_BAD")
	}
	if target.Value != "not-a-number" {
		t.Errorf("Value: got %q, want %q", target.Value, "not-a-number")
	}
	if target.Unwrap() == nil {
		t.Error("Unwrap() should return the underlying strconv error")
	}
}

func TestLoad_InvalidBoolValue(t *testing.T) {
	type Config struct {
		Debug bool `env:"TEST_DEBUG_BAD"`
	}
	setenv(t, "TEST_DEBUG_BAD", "yes-please")

	var cfg Config
	err := Load(&cfg)

	if _, ok := errors.AsType[*EnvValueParseError](err); !ok {
		t.Fatalf("expected EnvValueParseError, got %T: %v", err, err)
	}
}

func TestLoad_InvalidUintValue(t *testing.T) {
	type Config struct {
		Workers uint `env:"TEST_WORKERS_BAD"`
	}
	setenv(t, "TEST_WORKERS_BAD", "-5")

	var cfg Config
	err := Load(&cfg)

	if _, ok := errors.AsType[*EnvValueParseError](err); !ok {
		t.Fatalf("expected EnvValueParseError, got %T: %v", err, err)
	}
}

func TestLoad_UnsupportedFieldType(t *testing.T) {
	type Config struct {
		Tags []string `env:"TEST_TAGS"`
	}
	setenv(t, "TEST_TAGS", "a,b,c")

	var cfg Config
	err := Load(&cfg)

	if _, ok := errors.AsType[*EnvValueParseError](err); !ok {
		t.Fatalf("expected EnvValueParseError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// Load – full realistic config
// ---------------------------------------------------------------------------

func TestLoad_FullConfig(t *testing.T) {
	type DatabaseConfig struct {
		Host string `env:"TEST_DB_HOST"`
		Name string `env:"TEST_DB_NAME"`
	}
	type Config struct {
		AppName    string `env:"TEST_APP_NAME"`
		AppVersion string `env:"TEST_APP_VERSION"`
		Port       int    `env:"TEST_PORT"`
		Debug      bool   `env:"TEST_DEBUG"`
		MaxRetries uint   `env:"TEST_MAX_RETRIES"`
		Timeout    *int   `env:"TEST_TIMEOUT"`
		LogLevel   string `env:"TEST_LOG_LEVEL,default=info"`
		Database   DatabaseConfig
	}

	setenv(t, "TEST_APP_NAME", "my-service")
	setenv(t, "TEST_APP_VERSION", "1.2.3")
	setenv(t, "TEST_PORT", "8080")
	setenv(t, "TEST_DEBUG", "false")
	setenv(t, "TEST_MAX_RETRIES", "3")
	setenv(t, "TEST_TIMEOUT", "30")
	setenv(t, "TEST_DB_HOST", "localhost")
	setenv(t, "TEST_DB_NAME", "mydb")
	// TEST_LOG_LEVEL intentionally not set — default should apply

	var cfg Config
	if err := Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppName != "my-service" {
		t.Errorf("AppName: got %q", cfg.AppName)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port: got %d", cfg.Port)
	}
	if cfg.Debug {
		t.Error("Debug: got true, want false")
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries: got %d", cfg.MaxRetries)
	}
	if cfg.Timeout == nil || *cfg.Timeout != 30 {
		t.Errorf("Timeout: got %v", cfg.Timeout)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want %q (default)", cfg.LogLevel, "info")
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host: got %q", cfg.Database.Host)
	}
	if cfg.Database.Name != "mydb" {
		t.Errorf("Database.Name: got %q", cfg.Database.Name)
	}
}
