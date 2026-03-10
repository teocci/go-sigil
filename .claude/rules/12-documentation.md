# Documentation

## Principle

**Basic documentation for guidance** - Enough to understand intent, not exhaustive prose.

## Package Documentation

```go
// Package handlers provides HTTP request handlers for the API.
//
// Handlers are responsible for:
//   - Parsing and validating request input
//   - Calling appropriate services
//   - Formatting HTTP responses
//
// All handlers follow RESTful conventions and return JSON responses.
package handlers
```

## Exported Types

```go
// UserService handles user-related business logic.
// It coordinates between the repository, cache, and event publisher.
type UserService struct {
    repo   UserRepository
    cache  Cache
    events EventPublisher
    config *config.AuthConfig
}

// User represents a registered user in the system.
type User struct {
    ID        string    `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    Name      string    `json:"name" db:"name"`
    Role      Role      `json:"role" db:"role"`
    CreatedAt time.Time `json:"createdAt" db:"created_at"`
    UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}
```

## Exported Functions

```go
// NewUserService creates a new UserService with the given dependencies.
func NewUserService(
    repo UserRepository,
    cache Cache,
    events EventPublisher,
    config *config.AuthConfig,
) *UserService {
    return &UserService{
        repo:   repo,
        cache:  cache,
        events: events,
        config: config,
    }
}

// GetByID retrieves a user by their unique identifier.
// Returns ErrNotFound if the user does not exist.
func (s *UserService) GetByID(ctx context.Context, id string) (*User, error) {
    // ...
}

// Create registers a new user with the given details.
// Returns ErrEmailExists if the email is already registered.
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
    // ...
}
```

## Interface Documentation

```go
// UserRepository defines the data access interface for users.
// Implementations must be safe for concurrent use.
type UserRepository interface {
    // GetByID retrieves a user by ID. Returns ErrNotFound if not found.
    GetByID(ctx context.Context, id string) (*User, error)

    // GetByEmail retrieves a user by email. Returns ErrNotFound if not found.
    GetByEmail(ctx context.Context, email string) (*User, error)

    // Save creates or updates a user.
    Save(ctx context.Context, user *User) error

    // Delete removes a user by ID. Returns ErrNotFound if not found.
    Delete(ctx context.Context, id string) error
}
```

## Error Documentation

```go
// Sentinel errors for user operations.
var (
    // ErrNotFound indicates the requested user was not found.
    ErrNotFound = errors.New("user not found")

    // ErrEmailExists indicates the email is already registered.
    ErrEmailExists = errors.New("email already exists")

    // ErrInvalidCredentials indicates invalid login credentials.
    ErrInvalidCredentials = errors.New("invalid credentials")
)
```

## Constants Documentation

```go
// User roles define permission levels in the system.
const (
    RoleUser  Role = "user"  // Standard user with basic permissions
    RoleAdmin Role = "admin" // Administrator with full access
)

// Pagination limits.
const (
    DefaultPageSize = 20  // Default items per page
    MaxPageSize     = 100 // Maximum items per page
)
```

## Inline Comments

```go
// Document why, not what
func (s *Service) Process(ctx context.Context, data []byte) error {
    // Use worker pool to limit concurrent external API calls
    // to avoid rate limiting (max 10 concurrent requests)
    sem := make(chan struct{}, 10)

    // Sort by priority descending to process important items first
    sort.Slice(items, func(i, j int) bool {
        return items[i].Priority > items[j].Priority
    })

    // Retry with exponential backoff for transient failures
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := s.externalAPI.Call(ctx, data)
        if err == nil {
            return nil
        }
        if !isRetryable(err) {
            return err
        }
        time.Sleep(backoff(attempt))
    }

    return ErrMaxRetriesExceeded
}
```

## API Documentation Comments

```go
// Handler for GET /users/:id
// Retrieves a user by their unique identifier.
//
// Path Parameters:
//   - id: User's unique identifier (UUID)
//
// Responses:
//   - 200: User found, returns user object
//   - 404: User not found
//   - 500: Internal server error
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
    // ...
}

// Handler for POST /users
// Creates a new user account.
//
// Request Body:
//   - email: Valid email address (required)
//   - password: Min 8 characters (required)
//   - name: User's display name (required)
//
// Responses:
//   - 201: User created, returns user object
//   - 400: Validation error
//   - 409: Email already exists
//   - 500: Internal server error
func (h *UserHandler) Create(c *fiber.Ctx) error {
    // ...
}
```

## README Structure

```markdown
# Project Name

Brief description of what the project does.

## Requirements

- Go 1.21+
- PostgreSQL 15+
- Redis 7+

## Getting Started

### Configuration

Copy `.env.example` to `.env` and configure:

\`\`\`bash
cp .env.example .env
\`\`\`

### Running

\`\`\`bash
# Development
make run

# Production
make build
./bin/server
\`\`\`

### Testing

\`\`\`bash
make test
make test-coverage
\`\`\`

## Project Structure

\`\`\`
cmd/           - Entry points
internal/      - Private packages
pkg/           - Public packages
api/           - API specifications
configs/       - Configuration files
\`\`\`

## API Documentation

See [API.md](./API.md) or run the server and visit `/docs`.
```

## Avoid

- Obvious comments (`// increment i` for `i++`)
- Commented-out code (use git)
- TODO without context or owner
- Outdated documentation (worse than none)
- Documenting unexported types (usually unnecessary)
