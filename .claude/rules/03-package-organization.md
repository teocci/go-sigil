# Package Organization

## Standard Project Layout

```
project/
├── cmd/
│   └── server/
│       └── main.go           # Entry point, minimal code
│
├── internal/                  # Private packages (not importable)
│   ├── config/               # Configuration loading
│   │   └── config.go
│   ├── handlers/             # HTTP handlers (Fiber)
│   │   ├── user.go
│   │   └── health.go
│   ├── middleware/           # HTTP middleware
│   │   ├── auth.go
│   │   └── logging.go
│   ├── services/             # Business logic
│   │   └── user.go
│   ├── repository/           # Data access
│   │   └── user.go
│   ├── models/               # Domain types
│   │   └── user.go
│   └── workers/              # Background jobs
│       └── processor.go
│
├── pkg/                       # Public packages (importable by others)
│   └── validator/
│       └── validator.go
│
├── api/                       # API specs, OpenAPI, protobuf
│   └── openapi.yaml
│
├── web/                       # Frontend assets
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── templates/
│
├── scripts/                   # Build, deploy scripts
│   └── migrate.sh
│
├── configs/                   # Configuration files
│   ├── config.yaml
│   └── config.example.yaml
│
├── .env.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Package Principles

### Single Responsibility

Each package has one clear purpose:

```go
// internal/repository/user.go - ONLY data access
type UserRepository struct {
    db *sqlx.DB
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error)
func (r *UserRepository) Save(ctx context.Context, user *models.User) error

// internal/services/user.go - ONLY business logic
type UserService struct {
    repo   *repository.UserRepository
    cache  cache.Cache
    events events.Publisher
}

func (s *UserService) Register(ctx context.Context, req RegisterRequest) (*models.User, error)
```

### Dependency Direction

Dependencies flow inward: handlers → services → repository

```
handlers (HTTP layer)
    ↓
services (business logic)
    ↓
repository (data access)
    ↓
models (domain types)
```

### Package Naming

```go
// Package name matches directory
// internal/handlers/user.go
package handlers

// Usage is clear
handlers.NewUserHandler(svc)

// Avoid redundant naming
// Bad: package userhandlers → userhandlers.UserHandler
// Good: package handlers → handlers.UserHandler
```

## The `internal/` Package

Everything under `internal/` cannot be imported from outside:

```go
// This works (same module)
import "myproject/internal/config"

// This fails (external module)
// import "github.com/someone/myproject/internal/config"
// Error: use of internal package not allowed
```

Use `internal/` for:
- Business logic
- Data access
- Configuration
- Handlers
- Anything not meant for external use

## The `pkg/` Package

Only for packages intended to be imported by other projects:

```go
// pkg/validator/validator.go
package validator

// Public, stable API
func ValidateEmail(email string) error
func ValidatePhone(phone string) error
```

**If you're not building a library, you probably don't need `pkg/`.**

## The `cmd/` Package

Entry points only. Minimal code:

```go
// cmd/server/main.go
package main

import (
    "log"
    "os"

    "myproject/internal/config"
    "myproject/internal/server"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    srv := server.New(cfg)
    if err := srv.Run(); err != nil {
        log.Fatalf("server error: %v", err)
        os.Exit(1)
    }
}
```

## Package Size

- **Too small**: 1-2 files with minimal functionality → merge with related package
- **Too large**: >15 files or >2000 lines → split by responsibility
- **Just right**: 3-10 files, cohesive functionality

## Circular Dependencies

Go forbids circular imports. Structure to avoid them:

```go
// Bad: handlers imports services, services imports handlers
// handlers/user.go
import "myproject/internal/services" // services.UserService

// services/user.go
import "myproject/internal/handlers" // CIRCULAR!

// Good: use interfaces to break the cycle
// services/user.go
type EventPublisher interface {
    Publish(ctx context.Context, event Event) error
}

type UserService struct {
    publisher EventPublisher // Interface, not concrete type
}
```

## File Organization Within Package

```go
// models/user.go - One type per file (for larger types)
package models

type User struct {
    ID        string
    Email     string
    Name      string
    CreatedAt time.Time
}

func (u *User) IsActive() bool { ... }
func (u *User) FullName() string { ... }
```

```go
// models/types.go - Small related types can share a file
package models

type Status string

const (
    StatusPending   Status = "pending"
    StatusActive    Status = "active"
    StatusCompleted Status = "completed"
)

type Pagination struct {
    Page  int
    Limit int
}
```
