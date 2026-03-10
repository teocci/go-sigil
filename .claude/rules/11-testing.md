# Testing

## Test File Convention

```
package/
├── user.go
└── user_test.go
```

## Test Function Naming

```go
// Function tests
func TestUserService_Create(t *testing.T) { }
func TestUserService_Create_WithInvalidEmail(t *testing.T) { }
func TestUserService_Create_WhenEmailExists(t *testing.T) { }

// Method tests
func TestUser_FullName(t *testing.T) { }
func TestUser_IsActive(t *testing.T) { }
```

## Table-Driven Tests

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {
            name:    "valid email",
            email:   "user@example.com",
            wantErr: false,
        },
        {
            name:    "missing @",
            email:   "userexample.com",
            wantErr: true,
        },
        {
            name:    "missing domain",
            email:   "user@",
            wantErr: true,
        },
        {
            name:    "empty string",
            email:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
            }
        })
    }
}
```

## Test Helpers

```go
// testutil/testutil.go
package testutil

import "testing"

func AssertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}

func AssertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func AssertError(t *testing.T, err error) {
    t.Helper()
    if err == nil {
        t.Fatal("expected error, got nil")
    }
}

func AssertErrorIs(t *testing.T, err, target error) {
    t.Helper()
    if !errors.Is(err, target) {
        t.Errorf("error = %v, want %v", err, target)
    }
}
```

## Mocking with Interfaces

```go
// internal/services/user_test.go
package services

import (
    "context"
    "testing"

    "myproject/internal/models"
)

// Mock implementation
type mockUserRepo struct {
    getByIDFunc   func(ctx context.Context, id string) (*models.User, error)
    saveFunc      func(ctx context.Context, user *models.User) error
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
    return m.getByIDFunc(ctx, id)
}

func (m *mockUserRepo) Save(ctx context.Context, user *models.User) error {
    return m.saveFunc(ctx, user)
}

func TestUserService_GetByID(t *testing.T) {
    expectedUser := &models.User{ID: "123", Name: "John"}

    repo := &mockUserRepo{
        getByIDFunc: func(ctx context.Context, id string) (*models.User, error) {
            if id == "123" {
                return expectedUser, nil
            }
            return nil, ErrNotFound
        },
    }

    service := NewUserService(repo, nil, nil, nil)

    t.Run("found", func(t *testing.T) {
        user, err := service.GetByID(context.Background(), "123")
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if user.ID != expectedUser.ID {
            t.Errorf("got user ID %s, want %s", user.ID, expectedUser.ID)
        }
    })

    t.Run("not found", func(t *testing.T) {
        _, err := service.GetByID(context.Background(), "999")
        if !errors.Is(err, ErrNotFound) {
            t.Errorf("got error %v, want %v", err, ErrNotFound)
        }
    })
}
```

## Integration Tests

```go
// internal/repository/user_test.go
// +build integration

package repository

import (
    "context"
    "testing"

    "myproject/internal/testutil"
)

func TestUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)

    repo := NewUserRepository(db)
    ctx := context.Background()

    t.Run("create and get", func(t *testing.T) {
        user := &models.User{
            Email: "test@example.com",
            Name:  "Test User",
        }

        err := repo.Save(ctx, user)
        if err != nil {
            t.Fatalf("save failed: %v", err)
        }

        got, err := repo.GetByID(ctx, user.ID)
        if err != nil {
            t.Fatalf("get failed: %v", err)
        }

        if got.Email != user.Email {
            t.Errorf("got email %s, want %s", got.Email, user.Email)
        }
    })
}
```

## HTTP Handler Tests

```go
// internal/handlers/user_test.go
package handlers

import (
    "encoding/json"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/gofiber/fiber/v2"
)

func TestUserHandler_GetByID(t *testing.T) {
    app := fiber.New()

    mockService := &mockUserService{
        getByIDFunc: func(ctx context.Context, id string) (*models.User, error) {
            if id == "123" {
                return &models.User{ID: "123", Name: "John"}, nil
            }
            return nil, ErrNotFound
        },
    }

    handler := NewUserHandler(mockService)
    app.Get("/users/:id", handler.GetByID)

    t.Run("found", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/users/123", nil)
        resp, err := app.Test(req)
        if err != nil {
            t.Fatalf("request failed: %v", err)
        }

        if resp.StatusCode != 200 {
            t.Errorf("got status %d, want 200", resp.StatusCode)
        }

        var body dto.Response
        json.NewDecoder(resp.Body).Decode(&body)

        if !body.Success {
            t.Error("expected success response")
        }
    })

    t.Run("not found", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/users/999", nil)
        resp, err := app.Test(req)
        if err != nil {
            t.Fatalf("request failed: %v", err)
        }

        if resp.StatusCode != 404 {
            t.Errorf("got status %d, want 404", resp.StatusCode)
        }
    })
}
```

## Test Fixtures

```go
// internal/testutil/fixtures.go
package testutil

import "myproject/internal/models"

func NewTestUser() *models.User {
    return &models.User{
        ID:    "test-user-id",
        Email: "test@example.com",
        Name:  "Test User",
    }
}

func NewTestBooking(userID string) *models.Booking {
    return &models.Booking{
        ID:     "test-booking-id",
        UserID: userID,
        Status: models.StatusPending,
    }
}
```

## Benchmarks

```go
func BenchmarkUserService_GetByID(b *testing.B) {
    service := setupTestService()
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.GetByID(ctx, "123")
    }
}

func BenchmarkHashPassword(b *testing.B) {
    password := "testpassword123"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        HashPassword(password)
    }
}
```

## Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose
go test -v ./...

# Specific package
go test -v ./internal/services/...

# Skip integration tests
go test -short ./...

# Run integration tests
go test -tags=integration ./...

# Run benchmarks
go test -bench=. ./...

# Race detector
go test -race ./...
```

## Test Organization

```
internal/
├── services/
│   ├── user.go
│   └── user_test.go           # Unit tests
├── repository/
│   ├── user.go
│   └── user_integration_test.go  # Integration tests
├── handlers/
│   ├── user.go
│   └── user_test.go           # HTTP tests
└── testutil/
    ├── db.go                  # Test database helpers
    ├── fixtures.go            # Test data
    └── assertions.go          # Custom assertions
```
