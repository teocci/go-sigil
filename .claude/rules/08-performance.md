# Performance

## Know the Cost

Every decision has performance implications. Understand them before choosing.

## Memory Allocation

### Preallocate Slices

```go
// Bad - grows slice multiple times
var result []Item
for _, v := range input {
    result = append(result, transform(v))
}

// Good - single allocation
result := make([]Item, 0, len(input))
for _, v := range input {
    result = append(result, transform(v))
}

// Best - if size is known
result := make([]Item, len(input))
for i, v := range input {
    result[i] = transform(v)
}
```

### Preallocate Maps

```go
// Bad - map grows as needed
m := make(map[string]int)

// Good - preallocate if size known
m := make(map[string]int, expectedSize)
```

### Use sync.Pool for Frequent Allocations

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func Process(data []byte) []byte {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer
    buf.Write(data)
    return buf.Bytes()
}
```

### Avoid Allocations in Hot Paths

```go
// Bad - allocates on every call
func (s *Service) FormatKey(prefix, id string) string {
    return prefix + ":" + id // Creates new string
}

// Better - use strings.Builder for complex cases
func (s *Service) FormatKey(prefix, id string) string {
    var b strings.Builder
    b.Grow(len(prefix) + 1 + len(id))
    b.WriteString(prefix)
    b.WriteByte(':')
    b.WriteString(id)
    return b.String()
}

// Best for simple cases - fmt.Sprintf with reused buffer
// Or just accept the small allocation if called rarely
```

## String Operations

### strings.Builder for Concatenation

```go
// Bad - O(n²) for many concatenations
var s string
for _, item := range items {
    s += item.Name + ", "
}

// Good - O(n)
var b strings.Builder
for i, item := range items {
    if i > 0 {
        b.WriteString(", ")
    }
    b.WriteString(item.Name)
}
s := b.String()
```

### Avoid String-to-Byte Conversions in Loops

```go
// Bad - converts on every iteration
for _, s := range strings {
    hash.Write([]byte(s))
}

// Good - convert once if reusing
data := []byte(input)
for range iterations {
    hash.Write(data)
}
```

## Struct Design

### Order Fields by Size (Reduce Padding)

```go
// Bad - 24 bytes due to padding
type Bad struct {
    a bool   // 1 byte + 7 padding
    b int64  // 8 bytes
    c bool   // 1 byte + 7 padding
}

// Good - 16 bytes
type Good struct {
    b int64  // 8 bytes
    a bool   // 1 byte
    c bool   // 1 byte + 6 padding
}
```

### Pointer vs Value Receivers

```go
// Use pointer receiver when:
// - Method modifies receiver
// - Struct is large (>64 bytes roughly)
// - Consistency with other methods

func (u *User) SetName(name string) { // Modifies
    u.Name = name
}

func (u *User) FullName() string { // Large struct, consistent
    return u.FirstName + " " + u.LastName
}

// Use value receiver when:
// - Struct is small
// - Method is a simple accessor
// - Immutability is desired

func (p Point) Distance(other Point) float64 { // Small, immutable
    dx := p.X - other.X
    dy := p.Y - other.Y
    return math.Sqrt(dx*dx + dy*dy)
}
```

## Control Flow Cost

### Switch vs If-Else

```go
// switch - compiler may optimize to jump table
switch status {
case "pending":
    return handlePending()
case "active":
    return handleActive()
case "completed":
    return handleCompleted()
}

// if-else - sequential checks
if status == "pending" {
    return handlePending()
} else if status == "active" {
    return handleActive()
}

// For string switches with many cases, consider map
var handlers = map[string]func() error{
    "pending":   handlePending,
    "active":    handleActive,
    "completed": handleCompleted,
}

if h, ok := handlers[status]; ok {
    return h()
}
```

### Range Over Index When Value Unused

```go
// Bad - copies value on each iteration
for _, item := range largeItems {
    count++
}

// Good - no copy
for range largeItems {
    count++
}

// Or with index
for i := range largeItems {
    if largeItems[i].IsActive {
        count++
    }
}
```

## Database Performance

### Use Prepared Statements

```go
// Prepare once, execute many times
stmt, err := db.PrepareContext(ctx, "SELECT * FROM users WHERE id = $1")
if err != nil {
    return err
}
defer stmt.Close()

for _, id := range userIDs {
    var user User
    err := stmt.QueryRowContext(ctx, id).Scan(&user.ID, &user.Name)
    // ...
}
```

### Batch Operations

```go
// Bad - N queries
for _, user := range users {
    _, err := db.ExecContext(ctx,
        "INSERT INTO users (name, email) VALUES ($1, $2)",
        user.Name, user.Email)
}

// Good - single query with batch
query := "INSERT INTO users (name, email) VALUES "
var values []interface{}
for i, user := range users {
    if i > 0 {
        query += ", "
    }
    query += fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
    values = append(values, user.Name, user.Email)
}
_, err := db.ExecContext(ctx, query, values...)
```

### Use Indexes

```sql
-- Index columns used in WHERE, JOIN, ORDER BY
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_bookings_user_date ON bookings(user_id, created_at);
```

## HTTP Performance

### Reuse HTTP Clients

```go
// Bad - creates new client per request
func fetch(url string) (*http.Response, error) {
    client := &http.Client{} // New client!
    return client.Get(url)
}

// Good - reuse client
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

func fetch(url string) (*http.Response, error) {
    return httpClient.Get(url)
}
```

### Stream Large Responses

```go
// Bad - buffer entire response
func (h *Handler) Export(c *fiber.Ctx) error {
    data, _ := h.service.GetAllData()
    return c.JSON(data) // Buffers everything
}

// Good - stream response
func (h *Handler) Export(c *fiber.Ctx) error {
    c.Set("Content-Type", "application/json")
    c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
        enc := json.NewEncoder(w)
        w.WriteString("[")
        first := true
        h.service.StreamData(func(item Item) {
            if !first {
                w.WriteString(",")
            }
            first = false
            enc.Encode(item)
        })
        w.WriteString("]")
    })
    return nil
}
```

## Profiling

### Enable Profiling

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // Main application
}
```

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Benchmarking

```go
func BenchmarkProcess(b *testing.B) {
    data := generateTestData()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        Process(data)
    }
}

func BenchmarkProcessAllocs(b *testing.B) {
    data := generateTestData()
    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        Process(data)
    }
}
```

```bash
go test -bench=. -benchmem ./...
```
