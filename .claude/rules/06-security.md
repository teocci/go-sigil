# Security First

## Secrets Management

### Never Commit Secrets

```gitignore
# .gitignore
.env
.env.*
!.env.example
*.pem
*.key
*_secret*
```

### Load from Environment

```go
// Secrets from environment only
type AuthConfig struct {
    JWTSecret string `env:"JWT_SECRET,required"`
}

// Never log secrets
func (c *Config) LogSafe() {
    log.Printf("config: host=%s port=%d", c.Server.Host, c.Server.Port)
    // Never log: c.Auth.JWTSecret, c.Database.Password
}
```

## Input Validation

### Validate All Input

```go
// internal/validator/validator.go
package validator

import "github.com/go-playground/validator/v10"

var validate = validator.New()

type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Password string `json:"password" validate:"required,min=8,max=72"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
}

func ValidateStruct(s interface{}) error {
    return validate.Struct(s)
}

// Usage in handler
func (h *Handler) Register(c *fiber.Ctx) error {
    var req RegisterRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
    }

    if err := validator.ValidateStruct(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error":   "validation failed",
            "details": formatValidationErrors(err),
        })
    }

    // Proceed with validated input
}
```

### Sanitize User Input

```go
import "github.com/microcosm-cc/bluemonday"

var policy = bluemonday.UGCPolicy()

func SanitizeHTML(input string) string {
    return policy.Sanitize(input)
}

// Before storing user-generated content
user.Bio = SanitizeHTML(req.Bio)
```

## SQL Injection Prevention

### Always Use Parameterized Queries

```go
// Good - parameterized
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := r.db.GetContext(ctx, &user,
        "SELECT * FROM users WHERE email = $1", email)
    return &user, err
}

// Good - named parameters with sqlx
func (r *UserRepo) Search(ctx context.Context, name string, status Status) ([]User, error) {
    query := `SELECT * FROM users WHERE name ILIKE :name AND status = :status`
    rows, err := r.db.NamedQueryContext(ctx, query, map[string]interface{}{
        "name":   "%" + name + "%",
        "status": status,
    })
    // ...
}

// NEVER - string concatenation
query := "SELECT * FROM users WHERE email = '" + email + "'" // DANGEROUS!
```

## Authentication

### Password Hashing

```go
import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12 // From config in production

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### JWT Handling

```go
import "github.com/golang-jwt/jwt/v5"

type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func (s *AuthService) GenerateToken(user *User) (string, error) {
    claims := Claims{
        UserID: user.ID,
        Role:   user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpiration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "myservice",
            Subject:   user.ID,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return []byte(s.cfg.JWTSecret), nil
    })

    if err != nil {
        return nil, fmt.Errorf("parse token: %w", err)
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    return claims, nil
}
```

## Rate Limiting

```go
import "github.com/gofiber/fiber/v2/middleware/limiter"

func SetupRateLimiter() fiber.Handler {
    return limiter.New(limiter.Config{
        Max:               100,
        Expiration:        1 * time.Minute,
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP()
        },
        LimitReached: func(c *fiber.Ctx) error {
            return c.Status(429).JSON(fiber.Map{
                "error": "too many requests",
            })
        },
    })
}

// Stricter for auth endpoints
func AuthRateLimiter() fiber.Handler {
    return limiter.New(limiter.Config{
        Max:        5,
        Expiration: 15 * time.Minute,
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP() + ":" + c.Path()
        },
    })
}
```

## HTTPS & Headers

```go
import "github.com/gofiber/fiber/v2/middleware/helmet"

func SetupSecurity(app *fiber.App) {
    // Security headers
    app.Use(helmet.New())

    // Custom security headers
    app.Use(func(c *fiber.Ctx) error {
        c.Set("X-Content-Type-Options", "nosniff")
        c.Set("X-Frame-Options", "DENY")
        c.Set("X-XSS-Protection", "1; mode=block")
        c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
        return c.Next()
    })
}
```

## CORS

```go
import "github.com/gofiber/fiber/v2/middleware/cors"

func SetupCORS(cfg *config.Config) fiber.Handler {
    return cors.New(cors.Config{
        AllowOrigins:     cfg.Server.AllowedOrigins, // From config
        AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
        AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
        AllowCredentials: true,
        MaxAge:           86400, // 24 hours
    })
}
```

## File Upload Security

```go
const (
    MaxUploadSize = 10 << 20 // 10 MB
    AllowedTypes  = "image/jpeg,image/png,image/webp"
)

func (h *Handler) UploadFile(c *fiber.Ctx) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "no file provided"})
    }

    // Check size
    if file.Size > MaxUploadSize {
        return c.Status(400).JSON(fiber.Map{"error": "file too large"})
    }

    // Check content type
    contentType := file.Header.Get("Content-Type")
    if !strings.Contains(AllowedTypes, contentType) {
        return c.Status(400).JSON(fiber.Map{"error": "invalid file type"})
    }

    // Generate safe filename
    ext := filepath.Ext(file.Filename)
    safeName := fmt.Sprintf("%s%s", uuid.NewString(), ext)

    // Save file
    if err := c.SaveFile(file, filepath.Join(uploadDir, safeName)); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "upload failed"})
    }

    return c.JSON(fiber.Map{"filename": safeName})
}
```

## Checklist

- [ ] Secrets in environment variables only
- [ ] .env files in .gitignore
- [ ] All input validated server-side
- [ ] Parameterized SQL queries only
- [ ] Passwords hashed with bcrypt (cost ≥12)
- [ ] JWT with short expiration
- [ ] Rate limiting on all endpoints
- [ ] Strict rate limiting on auth endpoints
- [ ] Security headers configured
- [ ] CORS properly configured
- [ ] File uploads validated (size, type)
- [ ] No sensitive data in logs
