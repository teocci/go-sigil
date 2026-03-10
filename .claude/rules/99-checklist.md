# Pre-Commit Checklist

## Analysis
- [ ] Requirements documented
- [ ] Package design defined
- [ ] Interface contracts identified
- [ ] Error scenarios mapped
- [ ] Context7 MCP consulted (if third-party APIs)

## Go Style
- [ ] `gofmt` / `goimports` applied
- [ ] Naming follows conventions (exported vs unexported)
- [ ] Imports grouped (stdlib, third-party, local)
- [ ] Early returns used (guard clauses)
- [ ] Receivers are short (single letter)

## Package Organization
- [ ] Single responsibility per package
- [ ] Dependencies flow inward
- [ ] No circular dependencies
- [ ] Private code in `internal/`
- [ ] File size reasonable (<500 lines)

## Error Handling
- [ ] All errors checked
- [ ] Errors wrapped with context (`%w`)
- [ ] Sentinel errors defined
- [ ] No `panic` except programmer errors
- [ ] Error messages lowercase, no punctuation

## No Hardcoding
- [ ] Secrets in environment variables
- [ ] Config struct with defaults
- [ ] Constants for magic numbers
- [ ] No hardcoded URLs/paths/ports

## Security
- [ ] Input validated
- [ ] SQL parameterized
- [ ] Passwords hashed (bcrypt)
- [ ] JWT properly configured
- [ ] Rate limiting in place
- [ ] Secrets not logged

## Concurrency
- [ ] Context propagated
- [ ] Goroutines have termination path
- [ ] Channels have clear ownership
- [ ] Mutex used correctly
- [ ] No data races (`go test -race`)

## Performance
- [ ] Slices preallocated when size known
- [ ] `strings.Builder` for concatenation
- [ ] HTTP client reused
- [ ] Database queries optimized
- [ ] No N+1 queries

## API Design
- [ ] RESTful resources (nouns, plural)
- [ ] RFC 6570 URI templates
- [ ] Proper HTTP status codes
- [ ] Consistent response format
- [ ] Pagination implemented

## Clean Architecture
- [ ] Layers separated (handler/service/repo)
- [ ] Dependencies injected
- [ ] Interfaces at consumer side
- [ ] Functions ≤30 lines
- [ ] No global mutable state

## Testing
- [ ] Unit tests for services
- [ ] Handler tests for API
- [ ] Table-driven tests used
- [ ] Mocks for external deps
- [ ] Tests pass (`go test ./...`)
- [ ] No race conditions (`go test -race`)

## Documentation
- [ ] Package docs present
- [ ] Exported functions documented
- [ ] Error conditions documented
- [ ] README up to date

## Final Checks
- [ ] `go build` succeeds
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes
- [ ] No TODO without owner/issue
- [ ] .env.example updated if needed
