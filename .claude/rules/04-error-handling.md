# Error Handling

## Core Principle

Errors are **values** in Go. Handle them explicitly—never ignore them.

## Basic Pattern

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("do something: %w", err)
}
```

## Error Wrapping (Go 1.13+)

Use `%w` to wrap errors for the error chain:

```go
// Wrap with context
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("find user %s: %w", id, err)
    }
    return user, nil
}

// Unwrap with errors.Is and errors.As
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrUserNotFound
}

var pgErr *pgconn.PgError
if errors.As(err, &pgErr) && pgErr.Code == "23505" {
    return nil, ErrDuplicateEmail
}
```

## Custom Errors

### Sentinel Errors

```go
// internal/errors/errors.go
package errors

import "errors"

// Sentinel errors - compare with errors.Is()
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrInvalidInput  = errors.New("invalid input")
    ErrAlreadyExists = errors.New("already exists")
)
```

### Typed Errors

```go
// For errors that carry additional data
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// Usage
func ValidateUser(u *User) error {
    if u.Email == "" {
        return &ValidationError{Field: "email", Message: "required"}
    }
    return nil
}

// Check with errors.As
var validErr *ValidationError
if errors.As(err, &validErr) {
    log.Printf("Invalid field: %s", validErr.Field)
}
```

### Error with HTTP Status

```go
// internal/errors/http.go
type HTTPError struct {
    Code    int
    Message string
    Err     error
}

func (e *HTTPError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *HTTPError) Unwrap() error {
    return e.Err
}

// Constructors
func NewBadRequest(msg string) *HTTPError {
    return &HTTPError{Code: 400, Message: msg}
}

func NewNotFound(msg string) *HTTPError {
    return &HTTPError{Code: 404, Message: msg}
}

func NewInternalError(err error) *HTTPError {
    return &HTTPError{Code: 500, Message: "internal error", Err: err}
}
```

## Error Handling in Layers

### Repository Layer

```go
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("query user: %w", err)
    }
    return &user, nil
}
```

### Service Layer

```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    if id == "" {
        return nil, ErrInvalidInput
    }

    user, err := s.repo.GetByID(ctx, id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, fmt.Errorf("user %s: %w", id, ErrNotFound)
        }
        return nil, fmt.Errorf("get user: %w", err)
    }

    return user, nil
}
```

### Handler Layer

```go
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    id := c.Params("id")

    user, err := h.service.GetUser(c.Context(), id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return c.Status(404).JSON(fiber.Map{"error": "user not found"})
        }
        if errors.Is(err, ErrInvalidInput) {
            return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
        }
        // Log internal errors, return generic message
        h.logger.Error("get user failed", "error", err, "id", id)
        return c.Status(500).JSON(fiber.Map{"error": "internal error"})
    }

    return c.JSON(user)
}
```

## Never Ignore Errors

```go
// Bad - error ignored
json.Unmarshal(data, &result)
file.Close()

// Good - handle or explicitly ignore with comment
if err := json.Unmarshal(data, &result); err != nil {
    return fmt.Errorf("unmarshal: %w", err)
}

// If truly ignorable (rare), document why
_ = file.Close() // Best effort cleanup, error already logged
```

## Panic vs Error

```go
// Use error for expected failures
func ParseConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    // ...
}

// Use panic ONLY for programmer errors (bugs)
func MustCompileRegex(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex %q: %v", pattern, err))
    }
    return re
}

// "Must" prefix indicates panic on error
var emailRegex = MustCompileRegex(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
```

## Error Messages

```go
// Good - lowercase, no punctuation, add context
return fmt.Errorf("parse config: %w", err)
return fmt.Errorf("user %s not found", id)
return fmt.Errorf("connect to %s:%d: %w", host, port, err)

// Bad - uppercase, punctuation, no context
return fmt.Errorf("Error: Failed to parse config.")
return err // No context added
```
