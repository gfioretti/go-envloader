# go-envloader

A lightweight Go library for populating structs from environment variables using struct tags.

## Installation

```sh
go get github.com/gfioretti/envloader
```

## Usage

Define a struct with `env` tags and call `envloader.Load`:

```go
import "github.com/gfioretti/envloader"

type Config struct {
    AppName  string `env:"APP_NAME"`
    Port     int    `env:"PORT,default=8080"`
    Debug    bool   `env:"DEBUG,optional"`
    Timeout  *int   `env:"TIMEOUT"`
}

func main() {
    var cfg Config
    if err := envloader.Load(&cfg); err != nil {
        log.Fatal(err)
    }
}
```

## Tag options

| Tag                        | Behaviour                                              |
|----------------------------|--------------------------------------------------------|
| `env:"KEY"`                | Required. Returns an error if the variable is not set. |
| `env:"KEY,optional"`       | Silently skipped if the variable is not set.           |
| `env:"KEY,default=VALUE"`  | Uses `VALUE` if the variable is not set.               |

## Supported field types

- `string`
- `bool` — accepts `true`, `false`, `1`, `0`, `TRUE`, `FALSE`, etc.
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- Pointers to any of the above (`*string`, `*int`, …) — set to `nil` when the variable is absent

Nested structs are traversed recursively. Unexported fields are silently ignored.

## Error types

All errors implement the standard `error` interface and can be inspected with `errors.As`:

| Type                  | Returned when                                              |
|-----------------------|------------------------------------------------------------|
| `DataTypeError`       | `Load` is called with a non-pointer or nil value           |
| `EnvTagMissingError`  | An exported field has no `env` tag                         |
| `EnvValueMissingError`| A required env var is not set                              |
| `EnvValueParseError`  | A value cannot be parsed into the field's type; `Unwrap()` returns the underlying `strconv` error |

```go
var err *envloader.EnvValueMissingError
if errors.As(err, &target) {
    fmt.Println("missing:", target.EnvVar)
}
```

## Nested structs

```go
type DatabaseConfig struct {
    Host string `env:"DB_HOST"`
    Port int    `env:"DB_PORT,default=5432"`
}

type Config struct {
    AppName  string         `env:"APP_NAME"`
    Database DatabaseConfig
}
```

## Running tests

```sh
go test ./...
```

## License

MIT
