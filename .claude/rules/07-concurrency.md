# Concurrency

## Core Principles

- **Don't communicate by sharing memory; share memory by communicating**
- Always use `context.Context` for cancellation and timeouts
- Goroutines must have a clear termination path
- Channels should have clear ownership

## Context Propagation

### Always Pass Context

```go
// Every function that does I/O should accept context
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    return s.repo.FindByID(ctx, id)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
    return &user, err
}

// HTTP handlers get context from request
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    ctx := c.Context() // or c.UserContext()
    user, err := h.service.GetUser(ctx, c.Params("id"))
    // ...
}
```

### Context with Timeout

```go
func (s *Service) ProcessWithTimeout(parentCtx context.Context, data []byte) error {
    ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
    defer cancel() // Always call cancel to release resources

    result, err := s.externalAPI.Call(ctx, data)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return fmt.Errorf("operation timed out")
        }
        return fmt.Errorf("api call: %w", err)
    }

    return s.processResult(ctx, result)
}
```

### Context Cancellation

```go
func (s *Service) LongRunningOperation(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            // Cleanup and return
            return ctx.Err()
        default:
            // Do work
            if err := s.doWork(); err != nil {
                return err
            }
        }
    }
}
```

## Goroutine Patterns

### Worker Pool

```go
func ProcessItems(ctx context.Context, items []Item, workers int) error {
    jobs := make(chan Item, len(items))
    results := make(chan error, len(items))

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for item := range jobs {
                select {
                case <-ctx.Done():
                    results <- ctx.Err()
                    return
                default:
                    results <- processItem(ctx, item)
                }
            }
        }()
    }

    // Send jobs
    for _, item := range items {
        jobs <- item
    }
    close(jobs)

    // Wait and close results
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect errors
    var errs []error
    for err := range results {
        if err != nil {
            errs = append(errs, err)
        }
    }

    if len(errs) > 0 {
        return fmt.Errorf("processing failed: %d errors", len(errs))
    }
    return nil
}
```

### Bounded Concurrency with errgroup

```go
import "golang.org/x/sync/errgroup"

func ProcessWithLimit(ctx context.Context, items []Item, limit int) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(limit) // Max concurrent goroutines

    for _, item := range items {
        item := item // Capture for goroutine
        g.Go(func() error {
            return processItem(ctx, item)
        })
    }

    return g.Wait()
}
```

### Fan-Out, Fan-In

```go
func FanOutFanIn(ctx context.Context, input <-chan Request) <-chan Response {
    const numWorkers = 5
    outputs := make([]<-chan Response, numWorkers)

    // Fan-out: distribute to workers
    for i := 0; i < numWorkers; i++ {
        outputs[i] = worker(ctx, input)
    }

    // Fan-in: merge results
    return merge(ctx, outputs...)
}

func worker(ctx context.Context, input <-chan Request) <-chan Response {
    output := make(chan Response)
    go func() {
        defer close(output)
        for req := range input {
            select {
            case <-ctx.Done():
                return
            case output <- process(req):
            }
        }
    }()
    return output
}

func merge(ctx context.Context, channels ...<-chan Response) <-chan Response {
    var wg sync.WaitGroup
    merged := make(chan Response)

    output := func(ch <-chan Response) {
        defer wg.Done()
        for resp := range ch {
            select {
            case <-ctx.Done():
                return
            case merged <- resp:
            }
        }
    }

    wg.Add(len(channels))
    for _, ch := range channels {
        go output(ch)
    }

    go func() {
        wg.Wait()
        close(merged)
    }()

    return merged
}
```

## Synchronization

### sync.Mutex

```go
type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *SafeCounter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}
```

### sync.RWMutex (Read-Heavy Workloads)

```go
type Cache struct {
    mu   sync.RWMutex
    data map[string]interface{}
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### sync.Once

```go
var (
    instance *Service
    once     sync.Once
)

func GetService() *Service {
    once.Do(func() {
        instance = &Service{}
        instance.init()
    })
    return instance
}
```

### sync.Pool

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func ProcessData(data []byte) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer
    buf.Write(data)
    return buf.String()
}
```

## Channel Patterns

### Channel Ownership

```go
// Creator owns the channel, creator closes it
func generator(ctx context.Context) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out) // Owner closes
        for i := 0; ; i++ {
            select {
            case <-ctx.Done():
                return
            case out <- i:
            }
        }
    }()
    return out
}
```

### Select with Default (Non-Blocking)

```go
select {
case msg := <-ch:
    process(msg)
default:
    // Channel not ready, do something else
}
```

### Timeout Pattern

```go
select {
case result := <-resultCh:
    return result, nil
case <-time.After(5 * time.Second):
    return nil, errors.New("timeout")
case <-ctx.Done():
    return nil, ctx.Err()
}
```

## Avoid

- Goroutines without termination path (leaks)
- Shared memory without synchronization
- Closing channels from receiver side
- `go func()` without error handling
- Unbuffered channels when buffered makes sense
- `time.Sleep` for synchronization (use channels)
