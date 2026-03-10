# Pre-Implementation Analysis

Before writing any code, create a brief analysis:

1. **Requirements** - What exactly needs to be built
2. **Package design** - Which packages will be affected/created
3. **Interface contracts** - What interfaces need to be defined
4. **Concurrency needs** - Does this require goroutines, channels, context?
5. **Error scenarios** - What can fail and how to handle it
6. **Performance impact** - Memory allocations, CPU cost, I/O patterns

## MCP Integration

If the implementation depends on third-party library/framework/API specifics, consult **Context7 MCP first** and base code/setup/config on retrieved docs.

Skip only if:
- Pure language/algorithm work
- User explicitly opts out

## Analysis Format

```markdown
## Analysis: [Feature Name]

### Requirements
- ...

### Packages Affected
- `internal/handlers/` - New endpoint
- `internal/services/` - Business logic
- `internal/models/` - New types

### New Packages
- `internal/workers/` - Background processing

### Interfaces
- `Processor` interface for pluggable processing

### Concurrency
- Worker pool for batch processing
- Context propagation for cancellation

### Error Scenarios
- Database connection failure → retry with backoff
- Invalid input → 400 Bad Request with details
- Timeout → 504 Gateway Timeout

### Performance Notes
- Use sync.Pool for frequent allocations
- Stream large responses, don't buffer
```
