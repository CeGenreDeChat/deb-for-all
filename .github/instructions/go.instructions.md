# Go Coding Instructions

## Naming Conventions

### Packages
- Short names, lowercase, no underscores or camelCase
- Package name must match the directory name
- Avoid generic names like `util`, `common`, `misc`

### Variables and Functions
- Use camelCase for unexported identifiers
- Use PascalCase for exported identifiers
- Short names for limited scope variables (e.g., `i`, `n`, `err`)
- Descriptive names for broader scope variables
- Prefix interfaces with the behavior they describe (e.g., `Reader`, `Writer`)

### Constants
- Use PascalCase for exported constants
- Use camelCase for unexported constants
- Group related constants with `const (...)`

## File Structure

### Organization
- One package per directory
- Main file named after the package's primary functionality
- Separate types, interfaces, and implementations logically
- Place types and interfaces at the top of the file

### Imports
- Group imports into three blocks separated by blank lines:
  1. Standard library
  2. External packages
  3. Internal project packages
- Avoid import aliases except for name conflicts
- Never use dot imports (`.`)

## Error Handling

### Returning Errors
- Always return errors as the last return value
- Use `fmt.Errorf()` with `%w` to wrap errors
- Prefix error messages with operation context
- Never ignore returned errors

### Validation
- Validate inputs at the beginning of functions
- Return early on error (early return pattern)
- Avoid deep nesting of conditions

## Documentation

### Comments
- Document all exported identifiers
- Documentation comments start with the element name
- Use complete sentences ending with a period
- Explain "why" rather than "what" in inline comments

### Function Comment Format
```
// FunctionName performs a specific action.
// It takes X as parameter and returns Y.
```

## Best Practices

### Design
- Prefer composition over inheritance
- Define small, focused interfaces
- Accept interfaces, return concrete types
- Use zero values meaningfully

### Performance
- Avoid unnecessary allocations in loops
- Pre-allocate slices and maps when size is known
- Use `strings.Builder` for multiple concatenations
- Prefer `[]byte` over `string` for frequent manipulations

### Concurrency
- Communicate by sharing memory via channels
- Use `sync.WaitGroup` to wait for multiple goroutines
- Protect shared data with `sync.Mutex` or `sync.RWMutex`
- Prefer channels for coordination, mutexes for state

### Resources
- Use `defer` to release resources (files, connections)
- Place `defer` immediately after resource acquisition
- Check close errors for write operations

## Code Style

### Formatting
- Use `gofmt` or `goimports` consistently
- Limit line length to 100-120 characters
- One statement per line

### Structs
- Align struct tags vertically
- Use named struct literals
- Omit fields with zero value when appropriate

### Functions
- Limit the number of parameters (prefer configuration structs)
- Use variadic parameters sparingly
- Use named return values only when it improves clarity
