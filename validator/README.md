# validator

Wraps [go-playground/validator](https://github.com/go-playground/validator) with:

- Translated messages (English out of the box)
- Field names resolved from `label`, then `json`/`query`/`param`/`form`/`header` tags
- Field paths reported using JSON keys (including nested structs and slice indices)
- Each `FieldError` reports its `Source` (body, query, param, form, or header)

## Usage

```go
import "github.com/akfaiz/go-kit/validator"

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" label:"Email"`
	Password string `json:"password" validate:"required,min=8" label:"Password"`
}

v := validator.New()

err := v.Validate(&RegisterRequest{Email: "not-an-email"})

var vErr *validator.ValidationError
if errors.As(err, &vErr) {
	for _, fieldErr := range *vErr {
		fmt.Println(fieldErr.Field, fieldErr.Message)
		// email Email must be a valid email address
	}
}
```

## Locale support

Pass a `ContextExtractor` to resolve the locale per request and call
`ValidateContext` instead of `Validate`:

```go
v := validator.New(validator.Config{
	ContextExtractor: func(ctx context.Context) (string, bool) {
		locale, ok := ctx.Value(localeKey{}).(string)
		return locale, ok
	},
})

err := v.ValidateContext(ctx, req)
```

## Custom locales

Only English is registered by default. Add or replace supported
locales with `Config.Locales`, using any `github.com/go-playground/locales/<lang>`
translator paired with its matching `github.com/go-playground/validator/v10/translations/<lang>`
package:

```go
import (
	"github.com/go-playground/locales/fr"
	frTranslations "github.com/go-playground/validator/v10/translations/fr"
)

v := validator.New(validator.Config{
	Locales: []validator.Locale{
		{Tag: "en", Translator: en.New(), RegisterTranslations: enTranslations.RegisterDefaultTranslations},
		{Tag: "fr", Translator: fr.New(), RegisterTranslations: frTranslations.RegisterDefaultTranslations},
	},
	DefaultLocale: "en", // used when ContextExtractor is unset or resolves to an unsupported locale
	ContextExtractor: func(ctx context.Context) (string, bool) {
		locale, ok := ctx.Value(localeKey{}).(string)
		return locale, ok
	},
})
```

`DefaultLocale` defaults to the first entry in `Locales` (or `"en"` when `Locales`
is left unset).

## Custom tags

```go
v := validator.New(validator.Config{
	CustomLabelTag:  "name",
	CustomJSONTag:   "json",
	CustomQueryTag:  "query",
	CustomParamTag:  "param",
	CustomFormTag:   "form",
	CustomHeaderTag: "header",
})
```

## Header validation

Fields tagged with `header` (or `Config.CustomHeaderTag`) are validated the same as
`json`/`query`/`param`/`form` fields:

```go
type ListRequest struct {
	Authorization string `header:"Authorization" validate:"required" label:"Authorization"`
	Page          int    `query:"page" validate:"required,min=1" label:"Page"`
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
Set `Config.FieldPathFormat` to render paths as [RFC 6901](https://datatracker.ietf.org/doc/html/rfc6901)
JSON Pointers instead, e.g. `"/phones/0/number"`:

```go
v := validator.New(validator.Config{
	FieldPathFormat: validator.FieldPathJSONPointer,
})
```

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
