# validator

Wraps [go-playground/validator](https://github.com/go-playground/validator) with:

- Translated messages (English out of the box)
- Field names resolved from `label`, then `json`/`query`/`param`/`form`/`header` tags
- Field paths reported using JSON keys (including nested structs and slice indices)
- Each `FieldError` reports its `Source` (body, query, param, form, or header)
- Custom per-field messages and display names via `Messages()`/`Attributes()` methods

## Usage

```go
import "github.com/akfaiz/go-kit/validator"

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

v := validator.New()

err := v.Validate(&RegisterRequest{Email: "not-an-email"})

var vErr *validator.ValidationError
if errors.As(err, &vErr) {
	for _, fieldErr := range *vErr {
		fmt.Println(fieldErr.Field, fieldErr.Message)
		// email email must be a valid email address
	}
}
```

## Locale support

Pass `WithContextExtractor` to resolve the locale per request and call
`ValidateContext` instead of `Validate`:

```go
v := validator.New(
	validator.WithContextExtractor(func(ctx context.Context) (string, bool) {
		locale, ok := ctx.Value(localeKey{}).(string)
		return locale, ok
	}),
)

err := v.ValidateContext(ctx, req)
```

## Custom locales

Only English is registered by default. Add or replace supported locales with
`WithLocales`, using any `github.com/go-playground/locales/<lang>` translator paired
with its matching `github.com/go-playground/validator/v10/translations/<lang>` package:

```go
import (
	"github.com/go-playground/locales/fr"
	frTranslations "github.com/go-playground/validator/v10/translations/fr"
)

v := validator.New(
	validator.WithLocales(
		validator.Locale{Tag: "en", Translator: en.New(), RegisterTranslations: enTranslations.RegisterDefaultTranslations},
		validator.Locale{Tag: "fr", Translator: fr.New(), RegisterTranslations: frTranslations.RegisterDefaultTranslations},
	),
	validator.WithDefaultLocale("en"), // used when the context extractor is unset or resolves to an unsupported locale
	validator.WithContextExtractor(func(ctx context.Context) (string, bool) {
		locale, ok := ctx.Value(localeKey{}).(string)
		return locale, ok
	}),
)
```

`WithDefaultLocale` defaults to the first locale passed to `WithLocales` (or `"en"`
when `WithLocales` is unset).

## Custom tags

```go
v := validator.New(
	validator.WithLabelTag("label"),
	validator.WithJSONTag("json"),
	validator.WithQueryTag("query"),
	validator.WithParamTag("param"),
	validator.WithFormTag("form"),
	validator.WithHeaderTag("header"),
)
```

A `label` tag always wins over the json/query/param/form/header tag chain for display purposes:

```go
type RegisterRequest struct {
	Email string `json:"email" validate:"required,email" label:"Email Address"`
}
// fieldErr.Field  == "email" (still resolved from json/query/param/form/header)
// fieldErr.Message == "Email Address is a required field"
```

## Custom messages and attributes

A validated value can implement `MessagesProvider` and/or `AttributesProvider` to override
error text without touching struct tags — similar to Laravel's `FormRequest::messages()`/
`FormRequest::attributes()`. Both are checked on the top-level value passed to
`Validate`/`ValidateContext`:

```go
type CreatePostRequest struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"required"`
}

// Messages overrides the translated message for a specific "<field>.<rule>" combination.
func (r CreatePostRequest) Messages() map[string]string {
	return map[string]string{
		"title.required": "A title is required",
		"body.required":  "A message is required",
	}
}

// Attributes overrides the display name substituted into the default translated message.
func (r CreatePostRequest) Attributes() map[string]string {
	return map[string]string{
		"email": "email address",
	}
}
```

Keys use the same dot-notation field path reported in `FieldError.Field` (independent of
`WithFieldPathFormat`), with `"*"` standing in for any slice/array index — e.g.
`"phones.*.number.required"` matches the `required` rule on every element of a `Phones` slice.
A `Messages()` entry wins outright for that field/rule; an `Attributes()` entry only replaces
the field's display name inside the default translated text — overriding the field's `label`
tag if it has one.

`Attributes()` also applies to the *other* field named in cross-field rules (`eqfield`,
`gtfield`, `ltfield`, etc.) — e.g. `validate:"eqfield=PasswordConfirmation"` renders
`PasswordConfirmation`'s own attribute or `label` tag, if either is set, instead of its raw
struct field name. Precedence for that field's display text is `Attributes()` > `label` tag >
json/query/param/form/header tag chain.

## Custom validation rules

`RegisterValidation`/`RegisterValidationCtx` wrap the underlying
`govalidator.Validate.RegisterValidation`/`RegisterValidationCtx` directly, so you can add your
own `validate:"..."` rules:

```go
import govalidator "github.com/go-playground/validator/v10"

v := validator.New()
err := v.RegisterValidation("notreserved", func(fl govalidator.FieldLevel) bool {
	return fl.Field().String() != "admin"
})

type UsernameRequest struct {
	Username string `json:"username" validate:"required,notreserved"`
}
```

`RegisterValidation` takes a `govalidator.Func`. `RegisterValidationCtx` takes a
`govalidator.FuncCtx` instead, receiving the `context.Context` passed to `ValidateContext`.

`RegisterCustomTypeFunc` wraps `govalidator.Validate.RegisterCustomTypeFunc`, letting you teach
the validator how to extract a comparable value from a custom type (e.g. a `sql.NullString`)
before rules like `required`/`email`/`eqfield` run against it:

```go
v.RegisterCustomTypeFunc(func(field reflect.Value) any {
	if ns, ok := field.Interface().(sql.NullString); ok && ns.Valid {
		return ns.String
	}
	return nil
}, sql.NullString{})
```

## Header validation

Fields tagged with `header` (or `WithHeaderTag`) are validated the same as
`json`/`query`/`param`/`form` fields:

```go
type ListRequest struct {
	Authorization string `header:"Authorization" validate:"required"`
	Page          int    `query:"page" validate:"required,min=1"`
}
```

## Field source

Every `FieldError` reports which part of the request its `Field` came from, via
`Source` — one of `SourceBody` (default), `SourceQuery`, `SourceParam`, `SourceForm`,
or `SourceHeader` — determined by whichever tag resolved that field's name:

```go
for _, fieldErr := range *vErr {
	fmt.Println(fieldErr.Field, fieldErr.Source, fieldErr.Message)
	// authorization header Authorization is a required field
}
```

`Source` is empty (`""`) for `FieldError` values built manually via `NewError`,
`NewErrors`, `Add`, or `Addf`.

## Field path format

`FieldError.Field` defaults to dot notation (`FieldPathDot`), e.g. `"phones[0].number"`.
Pass `WithFieldPathFormat` to render paths as [RFC 6901](https://datatracker.ietf.org/doc/html/rfc6901)
JSON Pointers instead, e.g. `"/phones/0/number"`:

```go
v := validator.New(validator.WithFieldPathFormat(validator.FieldPathJSONPointer))
```

## Performance

`New` is relatively expensive (it builds the underlying `govalidator.Validate` and registers
locale translations) — construct one `*Validate` and reuse it, e.g. as a package-level or
injected singleton, rather than calling `New` per request. `*Validate` is safe for concurrent
use.

On the failure path, `Validate`/`ValidateContext` memoize their reflection-based field-path
resolution per `(struct type, govalidator namespace)` in an internal cache that lives on the
`*Validate` instance. Repeat validation failures on the same field of the same struct type —
the common case for a shared, long-lived `Validate` — skip the reflect walk entirely after the
first occurrence. The cache is never evicted, so a struct that fails validation across a very
large number of distinct slice indices will grow it proportionally; this is a non-issue for
ordinary request DTOs.

Benchmarks live in `validator_bench_test.go`:

```sh
go test ./validator/... -run '^$' -bench . -benchmem
```

## `Option`

`New(opts ...Option)` is configured via functional options:

| Option | Description |
| --- | --- |
| `WithContextExtractor(ContextExtractor)` | Resolve the active locale from a `context.Context` |
| `WithLabelTag(string)` | Override the label tag (default `"label"`) |
| `WithJSONTag(string)` | Override the JSON tag (default `"json"`) |
| `WithQueryTag(string)` | Override the query tag (default `"query"`) |
| `WithParamTag(string)` | Override the path param tag (default `"param"`) |
| `WithFormTag(string)` | Override the form tag (default `"form"`) |
| `WithHeaderTag(string)` | Override the header tag (default `"header"`) |
| `WithLocales(...Locale)` | Override the set of supported locales |
| `WithDefaultLocale(string)` | Override the fallback locale |
| `WithFieldPathFormat(FieldPathFormat)` | Override `FieldError.Field` rendering |

## `ValidationError`

`Validate`/`ValidateContext` return a `*ValidationError` (a `[]FieldError`) when
validation fails:

| Method | Description |
| --- | --- |
| `NewError(field, message string) *ValidationError` | Build with a single field error |
| `NewErrors(fieldErrors ...FieldError) *ValidationError` | Build from multiple field errors |
| `(*ValidationError) Add(field, message string) *ValidationError` | Append an error |
| `(*ValidationError) Addf(field, format string, args ...any) *ValidationError` | Append a formatted error |
| `(ValidationError) First() *FieldError` | First error, or `nil` if empty |
| `(*ValidationError) Fields() []string` | All field names |
| `(*ValidationError) Messages() []string` | All messages |
| `(ValidationError) Error() string` | Implements `error` |
| `(ValidationError) Errors() []FieldError` | Underlying slice |

`FieldError.Source` is one of: `SourceBody`, `SourceQuery`, `SourceParam`, `SourceForm`, `SourceHeader`.
