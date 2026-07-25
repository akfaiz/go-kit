# problem

Structured error responses based on [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807) (Problem Details for HTTP APIs).

## Usage

```go
import "github.com/akfaiz/go-kit/problem"

var ErrNotFound = problem.Register("Not Found", "not_found", 404)

func handler() error {
	return ErrNotFound("The requested resource was not found.").
		WithInstance("/users/123")
}
```

Errors are chainable and JSON-serializable:

```go
err := problem.New("Bad Request", "bad_request", 400).
	WithDetail("email is invalid").
	WithErrors(map[string]string{"email": "must be a valid email"}).
	WithCause(originalErr)

err.Error()          // "Bad Request: email is invalid"
errors.Unwrap(err)    // originalErr, via WithCause
```

`Register` and `New` return a distinct `*Error` on every call, so `errors.Is` matches by
`Type` rather than pointer identity — `errors.Is(err, ErrNotFound())` works regardless of
`Detail`/`Instance`/cause:

```go
if errors.Is(err, ErrNotFound()) {
	// handle not-found
}
```

## Stack traces

Like [github.com/pkg/errors](https://github.com/pkg/errors), `New` (and therefore
`Register`) captures the call stack at creation time. Print it with `%+v`, or walk it via
`StackTrace()`:

```go
err := problem.New("Internal Server Error", "internal_error", 500)

fmt.Printf("%+v", err)
// Internal Server Error
// main.handler
//     /app/main.go:42
// ...

for _, frame := range err.StackTrace() {
	fmt.Println(frame) // "/app/main.go:42 main.handler"
}
```

`%s`, `%v`, and `%q` print just the error message; only `%+v` includes the stack.

## API

| Method | Description |
| --- | --- |
| `Register(title, typeName string, status int, detail ...string) ErrorFunc` | Pre-register a reusable error factory |
| `New(title, typeName string, status int, detail ...string) *Error` | Build a one-off error |
| `(*Error) WithDetail(string) *Error` | Set `Detail` |
| `(*Error) WithErrors(any) *Error` | Attach structured error details (e.g. validation errors) |
| `(*Error) WithCause(error) *Error` | Set wrapped cause for `errors.Is`/`errors.As` |
| `(*Error) Is(target error) bool` | Matches `target` by `Type`, so `errors.Is` works across instances |
| `(*Error) WithInstance(string) *Error` | Set `Instance` (RFC 7807 URI) |
| `(*Error) StackTrace() []Frame` | Call stack captured when the error was created |
| `(*Error) Format(fmt.State, rune)` | Implements `fmt.Formatter`; `%+v` includes the stack trace |
