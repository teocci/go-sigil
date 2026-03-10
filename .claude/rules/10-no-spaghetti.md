# No Spaghetti Code

## Core Principles

- **Single Responsibility** - Each package/type/function has one job
- **Explicit Dependencies** - Inject via constructors, no globals
- **Small Interfaces** - Accept interfaces, return structs
- **Testability** - Design for easy testing

## Layered Architecture

```
┌─────────────────────────────────────────┐
│           Handlers (HTTP)               │  ← Parse request, validate, call service
├─────────────────────────────────────────┤
│           Services (Business)           │  ← Business logic, orchestration
├─────────────────────────────────────────┤
│         Repository (Data Access)        │  ← Database operations
├─────────────────────────────────────────┤
│            Models (Domain)              │  ← Types, entities
└─────────────────────────────────────────┘
```

### Layer Rules

| Layer | Can Import | Responsibilities |
|-------|------------|------------------|
| Handlers | Services, DTOs, Models | Parse HTTP, validate, format response |
| Services | Repository, Models | Business logic, validation, orchestration |
| Repository | Models | Data access only |
| Models | Nothing | Domain types |

## Dependency Injection

### Constructor Injection

```go
// internal/services/user.go
type UserService struct {
    repo   UserRepository
    cache  Cache
    events EventPublisher
    config *config.AuthConfig
}

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
```

### Interface Dependencies

```go
// Define interfaces where they're used (consumer side)
// internal/services/user.go

type UserRepository interface {
    GetByID(ctx context.Context, id string) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Save(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id string) error
}

type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

type EventPublisher interface {
    Publish(ctx context.Context, event Event) error
}
```

### Wire Up in Main

```go
// cmd/server/main.go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Initialize dependencies
    db := database.New(cfg.Database)
    cache := redis.New(cfg.Redis)
    events := kafka.NewPublisher(cfg.Kafka)

    // Create repositories
    userRepo := repository.NewUserRepository(db)
    bookingRepo := repository.NewBookingRepository(db)

    // Create services
    userService := services.NewUserService(userRepo, cache, events, &cfg.Auth)
    bookingService := services.NewBookingService(bookingRepo, userService)

    // Create handlers
    handlers := handlers.New(userService, bookingService)

    // Setup server
    srv := server.New(cfg.Server, handlers)
    srv.Run()
}
```

## Small Interfaces

```go
// Good - small, focused interfaces
type Reader interface {
    Read(ctx context.Context, id string) (*Entity, error)
}

type Writer interface {
    Write(ctx context.Context, entity *Entity) error
}

type Deleter interface {
    Delete(ctx context.Context, id string) error
}

// Compose when needed
type ReadWriter interface {
    Reader
    Writer
}

// Bad - large interface
type Repository interface {
    GetByID(ctx context.Context, id string) (*Entity, error)
    GetByEmail(ctx context.Context, email string) (*Entity, error)
    GetAll(ctx context.Context) ([]*Entity, error)
    GetByFilters(ctx context.Context, filters Filters) ([]*Entity, error)
    Save(ctx context.Context, entity *Entity) error
    Update(ctx context.Context, entity *Entity) error
    Delete(ctx context.Context, id string) error
    Count(ctx context.Context) (int, error)
    // ... 10 more methods
}
```

## Function Size

### Target: ≤30 Lines

```go
// Bad - too long, multiple responsibilities
func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
    // 100+ lines doing:
    // - Validation
    // - Database queries
    // - External API calls
    // - Email sending
    // - Logging
    // - Error handling for each
}

// Good - orchestrates smaller functions
func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
    order, err := s.validateAndGetOrder(ctx, orderID)
    if err != nil {
        return fmt.Errorf("validate order: %w", err)
    }

    if err := s.reserveInventory(ctx, order); err != nil {
        return fmt.Errorf("reserve inventory: %w", err)
    }

    if err := s.processPayment(ctx, order); err != nil {
        s.releaseInventory(ctx, order) // Compensate
        return fmt.Errorf("process payment: %w", err)
    }

    if err := s.notifyCustomer(ctx, order); err != nil {
        s.logger.Error("failed to notify customer", "order", orderID, "error", err)
        // Non-critical, don't fail the order
    }

    return nil
}
```

## No Global State

```go
// Bad - global mutable state
var db *sql.DB
var config *Config

func init() {
    db = connectDB()
    config = loadConfig()
}

func GetUser(id string) (*User, error) {
    return db.Query(...) // Hidden dependency
}

// Good - explicit dependencies
type UserService struct {
    db     *sql.DB
    config *Config
}

func NewUserService(db *sql.DB, config *Config) *UserService {
    return &UserService{db: db, config: config}
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    return s.db.QueryContext(ctx, ...) // Explicit dependency
}
```

## Error Handling Consistency

```go
// Consistent pattern across layers

// Repository - wrap with context
func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
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

// Service - add business context
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user %s: %w", id, err)
    }
    return user, nil
}

// Handler - map to HTTP response
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    user, err := h.service.GetUser(c.Context(), c.Params("id"))
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return c.Status(404).JSON(dto.Error("NOT_FOUND", "user not found"))
        }
        h.logger.Error("get user failed", "error", err)
        return c.Status(500).JSON(dto.Error("INTERNAL", "internal error"))
    }
    return c.JSON(dto.OK(user))
}
```

## Avoid

- Package-level variables (except constants)
- `init()` functions (make initialization explicit)
- Circular dependencies
- God structs with too many fields/methods
- Functions longer than 50 lines
- More than 3 levels of nesting
- Hidden dependencies (access globals inside functions)
- Interface pollution (too many methods)
