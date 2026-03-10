# No Hardcoding

## Never Hardcode

- Environment-specific values (URLs, hosts, ports)
- Secrets (API keys, passwords, tokens)
- Magic numbers
- Timeout values
- File paths
- Resource limits

## Constants for Fixed Values

```go
// internal/constants/constants.go
package constants

import "time"

// Application
const (
    AppName    = "myservice"
    AppVersion = "1.0.0"
)

// Limits
const (
    MaxPageSize     = 100
    DefaultPageSize = 20
    MaxUploadSize   = 10 << 20 // 10 MB
)

// Timeouts
const (
    DefaultTimeout     = 30 * time.Second
    DatabaseTimeout    = 10 * time.Second
    ShutdownTimeout    = 15 * time.Second
)

// Retry
const (
    MaxRetries   = 3
    RetryBackoff = 100 * time.Millisecond
)
```

```go
// internal/constants/status.go
package constants

type Status string

const (
    StatusPending   Status = "pending"
    StatusActive    Status = "active"
    StatusCompleted Status = "completed"
    StatusCancelled Status = "cancelled"
)

func (s Status) IsValid() bool {
    switch s {
    case StatusPending, StatusActive, StatusCompleted, StatusCancelled:
        return true
    }
    return false
}
```

## Configuration Structure

```go
// internal/config/config.go
package config

import (
    "time"

    "github.com/caarlos0/env/v9"
    "github.com/joho/godotenv"
)

type Config struct {
    App      AppConfig
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Auth     AuthConfig
}

type AppConfig struct {
    Name        string `env:"APP_NAME" envDefault:"myservice"`
    Environment string `env:"APP_ENV" envDefault:"development"`
    Debug       bool   `env:"APP_DEBUG" envDefault:"false"`
}

type ServerConfig struct {
    Host         string        `env:"SERVER_HOST" envDefault:"0.0.0.0"`
    Port         int           `env:"SERVER_PORT" envDefault:"8080"`
    ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
    WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"30s"`
}

type DatabaseConfig struct {
    Host         string        `env:"DB_HOST" envDefault:"localhost"`
    Port         int           `env:"DB_PORT" envDefault:"5432"`
    User         string        `env:"DB_USER,required"`
    Password     string        `env:"DB_PASSWORD,required"`
    Database     string        `env:"DB_NAME,required"`
    SSLMode      string        `env:"DB_SSL_MODE" envDefault:"disable"`
    MaxOpenConns int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
    MaxIdleConns int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
    MaxLifetime  time.Duration `env:"DB_MAX_LIFETIME" envDefault:"5m"`
}

type RedisConfig struct {
    Host     string `env:"REDIS_HOST" envDefault:"localhost"`
    Port     int    `env:"REDIS_PORT" envDefault:"6379"`
    Password string `env:"REDIS_PASSWORD"`
    DB       int    `env:"REDIS_DB" envDefault:"0"`
}

type AuthConfig struct {
    JWTSecret     string        `env:"JWT_SECRET,required"`
    JWTExpiration time.Duration `env:"JWT_EXPIRATION" envDefault:"24h"`
    BCryptCost    int           `env:"BCRYPT_COST" envDefault:"12"`
}

func Load() (*Config, error) {
    // Load .env file if exists (development)
    _ = godotenv.Load()

    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    return cfg, nil
}

// Helper methods
func (c *DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
    )
}

func (c *ServerConfig) Addr() string {
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
```

## Environment Files

```bash
# .env.example (commit this)
APP_NAME=myservice
APP_ENV=development
APP_DEBUG=true

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=30s

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=change_me
DB_NAME=myservice
DB_SSL_MODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379

JWT_SECRET=change_me_in_production
JWT_EXPIRATION=24h
```

```gitignore
# .gitignore
.env
.env.local
.env.*.local
```

## Configuration Validation

```go
func (c *Config) Validate() error {
    if c.App.Environment == "production" {
        if c.App.Debug {
            return errors.New("debug must be disabled in production")
        }
        if c.Database.SSLMode == "disable" {
            return errors.New("database SSL required in production")
        }
        if len(c.Auth.JWTSecret) < 32 {
            return errors.New("JWT secret must be at least 32 characters")
        }
    }
    return nil
}
```

## Dependency Injection

Pass configuration via constructors, never access globals:

```go
// Good - explicit dependency
type UserService struct {
    repo   UserRepository
    config *config.AuthConfig
}

func NewUserService(repo UserRepository, cfg *config.AuthConfig) *UserService {
    return &UserService{
        repo:   repo,
        config: cfg,
    }
}

// Bad - global access
type UserService struct {
    repo UserRepository
}

func (s *UserService) HashPassword(password string) (string, error) {
    return bcrypt.GenerateFromPassword([]byte(password), globalConfig.BCryptCost) // Bad!
}
```

## Feature Flags

```go
// internal/config/features.go
type Features struct {
    EnableNewSearch bool `env:"FEATURE_NEW_SEARCH" envDefault:"false"`
    EnableMetrics   bool `env:"FEATURE_METRICS" envDefault:"true"`
    EnableRateLimit bool `env:"FEATURE_RATE_LIMIT" envDefault:"true"`
}

// Usage
if cfg.Features.EnableNewSearch {
    router.Get("/search", h.NewSearchHandler)
} else {
    router.Get("/search", h.LegacySearchHandler)
}
```

## Avoid

```go
// Bad - hardcoded values
timeout := 30 * time.Second
maxSize := 10485760
dbHost := "localhost:5432"

// Good - from config or constants
timeout := cfg.Server.ReadTimeout
maxSize := constants.MaxUploadSize
dbHost := cfg.Database.Addr()
```
