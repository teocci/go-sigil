# API Design (Fiber + RFC 6570)

## URI Template Standard (RFC 6570)

Follow [RFC 6570](https://www.rfc-editor.org/rfc/rfc6570.html) for URI design:

### Path Parameters

```
/users/{id}
/users/{userId}/bookings/{bookingId}
/organizations/{orgId}/members/{memberId}
```

### Query Parameters

```
/users{?page,limit,sort}
/search{?q,category,minPrice,maxPrice}
/reports{?startDate,endDate,format}
```

### Path Segments

```
/users/{id}/profile
/users/{id}/settings
/users/{id}/bookings
```

## RESTful Resource Naming

### Resources (Nouns, Plural)

```go
// Good
GET    /users
GET    /users/{id}
POST   /users
PUT    /users/{id}
DELETE /users/{id}

// Bad
GET    /getUsers
GET    /user/{id}      // Should be plural
POST   /createUser     // Verb in path
DELETE /deleteUser/{id}
```

### Nested Resources

```go
// User's bookings
GET    /users/{userId}/bookings
GET    /users/{userId}/bookings/{bookingId}
POST   /users/{userId}/bookings

// Organization members
GET    /organizations/{orgId}/members
POST   /organizations/{orgId}/members/{userId}
DELETE /organizations/{orgId}/members/{userId}
```

### Actions (When REST Doesn't Fit)

```go
// Use verbs for actions that aren't CRUD
POST /users/{id}/verify-email
POST /bookings/{id}/cancel
POST /payments/{id}/refund
POST /auth/login
POST /auth/logout
POST /auth/refresh
```

## Fiber Router Setup

```go
// internal/server/routes.go
package server

import (
    "github.com/gofiber/fiber/v2"

    "myproject/internal/handlers"
    "myproject/internal/middleware"
)

func SetupRoutes(app *fiber.App, h *handlers.Handlers, mw *middleware.Middleware) {
    // API versioning
    api := app.Group("/api/v1")

    // Public routes
    api.Post("/auth/login", h.Auth.Login)
    api.Post("/auth/register", h.Auth.Register)
    api.Post("/auth/refresh", h.Auth.RefreshToken)

    // Protected routes
    protected := api.Group("", mw.RequireAuth)

    // Users
    users := protected.Group("/users")
    users.Get("/", h.User.List)           // GET /api/v1/users{?page,limit}
    users.Get("/:id", h.User.GetByID)     // GET /api/v1/users/{id}
    users.Put("/:id", h.User.Update)      // PUT /api/v1/users/{id}
    users.Delete("/:id", h.User.Delete)   // DELETE /api/v1/users/{id}

    // User's bookings (nested resource)
    users.Get("/:userId/bookings", h.Booking.ListByUser)
    users.Post("/:userId/bookings", h.Booking.Create)

    // Bookings
    bookings := protected.Group("/bookings")
    bookings.Get("/:id", h.Booking.GetByID)
    bookings.Put("/:id", h.Booking.Update)
    bookings.Delete("/:id", h.Booking.Delete)
    bookings.Post("/:id/cancel", h.Booking.Cancel) // Action

    // Search (query parameters)
    api.Get("/search", h.Search.Search) // GET /api/v1/search{?q,category,page,limit}
}
```

## Request/Response Patterns

### Request DTOs

```go
// internal/handlers/dto/user.go
package dto

type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
}

type UpdateUserRequest struct {
    Name  *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
    Phone *string `json:"phone,omitempty" validate:"omitempty,e164"`
}

type ListUsersRequest struct {
    Page   int    `query:"page" validate:"omitempty,min=1"`
    Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
    Sort   string `query:"sort" validate:"omitempty,oneof=name email created_at"`
    Order  string `query:"order" validate:"omitempty,oneof=asc desc"`
    Search string `query:"search" validate:"omitempty,max=100"`
}
```

### Response DTOs

```go
// internal/handlers/dto/response.go
package dto

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type ErrorInfo struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

type Meta struct {
    Page       int `json:"page"`
    Limit      int `json:"limit"`
    Total      int `json:"total"`
    TotalPages int `json:"totalPages"`
}

// Helper constructors
func OK(data interface{}) Response {
    return Response{Success: true, Data: data}
}

func OKWithMeta(data interface{}, meta *Meta) Response {
    return Response{Success: true, Data: data, Meta: meta}
}

func Error(code, message string) Response {
    return Response{Success: false, Error: &ErrorInfo{Code: code, Message: message}}
}

func ValidationError(details map[string]string) Response {
    return Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    "VALIDATION_ERROR",
            Message: "validation failed",
            Details: details,
        },
    }
}
```

## Handler Implementation

```go
// internal/handlers/user.go
package handlers

import (
    "github.com/gofiber/fiber/v2"

    "myproject/internal/handlers/dto"
    "myproject/internal/services"
)

type UserHandler struct {
    service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
    return &UserHandler{service: service}
}

// GET /users
func (h *UserHandler) List(c *fiber.Ctx) error {
    var req dto.ListUsersRequest
    if err := c.QueryParser(&req); err != nil {
        return c.Status(400).JSON(dto.Error("INVALID_QUERY", "invalid query parameters"))
    }

    // Apply defaults
    if req.Page == 0 {
        req.Page = 1
    }
    if req.Limit == 0 {
        req.Limit = 20
    }

    users, total, err := h.service.List(c.Context(), req)
    if err != nil {
        return c.Status(500).JSON(dto.Error("INTERNAL_ERROR", "failed to list users"))
    }

    return c.JSON(dto.OKWithMeta(users, &dto.Meta{
        Page:       req.Page,
        Limit:      req.Limit,
        Total:      total,
        TotalPages: (total + req.Limit - 1) / req.Limit,
    }))
}

// GET /users/:id
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
    id := c.Params("id")
    if id == "" {
        return c.Status(400).JSON(dto.Error("INVALID_ID", "user id required"))
    }

    user, err := h.service.GetByID(c.Context(), id)
    if err != nil {
        if errors.Is(err, services.ErrUserNotFound) {
            return c.Status(404).JSON(dto.Error("NOT_FOUND", "user not found"))
        }
        return c.Status(500).JSON(dto.Error("INTERNAL_ERROR", "failed to get user"))
    }

    return c.JSON(dto.OK(user))
}

// POST /users
func (h *UserHandler) Create(c *fiber.Ctx) error {
    var req dto.CreateUserRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(dto.Error("INVALID_BODY", "invalid request body"))
    }

    if errors := validateStruct(req); len(errors) > 0 {
        return c.Status(400).JSON(dto.ValidationError(errors))
    }

    user, err := h.service.Create(c.Context(), req)
    if err != nil {
        if errors.Is(err, services.ErrEmailExists) {
            return c.Status(409).JSON(dto.Error("CONFLICT", "email already exists"))
        }
        return c.Status(500).JSON(dto.Error("INTERNAL_ERROR", "failed to create user"))
    }

    return c.Status(201).JSON(dto.OK(user))
}
```

## HTTP Status Codes

| Status | When to Use |
|--------|-------------|
| 200 | Success (GET, PUT, PATCH) |
| 201 | Created (POST) |
| 204 | No Content (DELETE) |
| 400 | Bad Request (validation, parsing) |
| 401 | Unauthorized (no/invalid auth) |
| 403 | Forbidden (auth valid, no permission) |
| 404 | Not Found |
| 409 | Conflict (duplicate, state conflict) |
| 422 | Unprocessable Entity (business rule violation) |
| 429 | Too Many Requests (rate limit) |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

## API Versioning

```go
// Version in URL path (recommended)
api := app.Group("/api/v1")
apiV2 := app.Group("/api/v2")

// Or via header
app.Use(func(c *fiber.Ctx) error {
    version := c.Get("API-Version", "v1")
    c.Locals("apiVersion", version)
    return c.Next()
})
```

## Pagination Pattern

```go
// Query: GET /users?page=2&limit=20
type Pagination struct {
    Page  int `query:"page" validate:"omitempty,min=1"`
    Limit int `query:"limit" validate:"omitempty,min=1,max=100"`
}

func (p *Pagination) Offset() int {
    return (p.Page - 1) * p.Limit
}

func (p *Pagination) ApplyDefaults() {
    if p.Page == 0 {
        p.Page = 1
    }
    if p.Limit == 0 {
        p.Limit = 20
    }
}
```

## Filtering Pattern

```go
// Query: GET /bookings?status=active&minDate=2024-01-01&maxDate=2024-12-31
type BookingFilters struct {
    Status  *string    `query:"status"`
    MinDate *time.Time `query:"minDate"`
    MaxDate *time.Time `query:"maxDate"`
    UserID  *string    `query:"userId"`
}
```
