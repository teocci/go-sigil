# Go Style & Conventions

## Core Philosophy

Go is **expressive, concise, clean, and efficient**. Write code that:
- Is explicit over implicit (no magic)
- Favors composition over inheritance
- Uses small interfaces
- Handles errors explicitly
- Leverages the type system

## Naming Conventions

### Packages

```go
// Short, lowercase, single-word preferred
package handlers   // Good
package httpHandlers // Bad - avoid camelCase
package http_handlers // Bad - avoid underscores
```

### Exported vs Unexported

```go
// Exported (public) - PascalCase
type UserService struct {}
func NewUserService() *UserService {}
const MaxRetries = 3

// Unexported (private) - camelCase
type userRepository struct {}
func parseToken(s string) {}
const defaultTimeout = 30 * time.Second
```

### Variables and Functions

```go
// Short names for small scopes
for i, v := range items {}
if err != nil {}

// Descriptive names for larger scopes
userRepository := NewUserRepository(db)
maxConcurrentWorkers := cfg.Workers.Max

// Receivers - short, typically first letter
func (s *UserService) GetByID(id string) (*User, error) {}
func (r *userRepository) findByEmail(email string) (*User, error) {}

// Avoid stuttering
user.Name          // Good
user.UserName      // Bad - stutters with type
```

### Interfaces

```go
// Single-method interfaces: method name + "er"
type Reader interface { Read(p []byte) (n int, err error) }
type Stringer interface { String() string }
type Processor interface { Process(ctx context.Context, data []byte) error }

// Multi-method interfaces: descriptive noun
type UserStore interface {
    Get(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

## Formatting

Always run `gofmt` or `goimports`. No exceptions.

```go
// Imports: stdlib, blank line, third-party, blank line, local
import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/jmoiron/sqlx"

    "myproject/internal/config"
    "myproject/internal/models"
)
```

## Zero Values

Leverage zero values for cleaner code:

```go
// Zero values are useful defaults
var count int        // 0
var name string      // ""
var enabled bool     // false
var users []User     // nil (valid for append, len, range)
var data map[string]int // nil (must initialize before write)

// Use zero values in structs
type Config struct {
    Host    string        // defaults to ""
    Port    int           // defaults to 0
    Timeout time.Duration // defaults to 0
}

// Check with zero value
if user.ID == "" {
    return errors.New("user ID required")
}
```

## Control Flow

### Early Returns (Guard Clauses)

```go
// Good - early returns, flat structure
func (s *Service) Process(ctx context.Context, id string) (*Result, error) {
    if id == "" {
        return nil, ErrInvalidID
    }

    user, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }

    if !user.IsActive {
        return nil, ErrUserInactive
    }

    return s.processUser(ctx, user)
}

// Bad - nested conditionals
func (s *Service) Process(ctx context.Context, id string) (*Result, error) {
    if id != "" {
        user, err := s.repo.Get(ctx, id)
        if err == nil {
            if user.IsActive {
                return s.processUser(ctx, user)
            } else {
                return nil, ErrUserInactive
            }
        } else {
            return nil, err
        }
    } else {
        return nil, ErrInvalidID
    }
}
```

### Switch vs If

```go
// Use switch for multiple conditions on same variable
switch status {
case StatusPending:
    return handlePending(ctx, item)
case StatusActive:
    return handleActive(ctx, item)
case StatusCompleted:
    return handleCompleted(ctx, item)
default:
    return nil, fmt.Errorf("unknown status: %s", status)
}

// Use if for complex boolean conditions
if user.IsAdmin && user.HasPermission(PermWrite) {
    // ...
}
```

## Avoid

- `panic` except for truly unrecoverable programmer errors
- `init()` functions (make initialization explicit)
- Global mutable state
- Package-level variables (except constants)
- Naked returns in functions longer than a few lines
- Empty interface (`interface{}`) without type assertion
