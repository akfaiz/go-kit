package validator

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	govalidator "github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
)

// Validate is a wrapper around the go-playground/validator that supports i18n error messages and custom field name resolution.
type Validate struct {
	validate         *govalidator.Validate
	uni              *ut.UniversalTranslator
	contextExtractor ContextExtractor
	defaultLocale    string
	fieldPathFormat  FieldPathFormat
	jsonTag          string
	queryTag         string
	paramTag         string
	formTag          string
}

// Locale registers a supported locale for translated validation messages.
type Locale struct {
	// Tag is the locale identifier returned by ContextExtractor (e.g. "en", "id", "fr").
	Tag string
	// Translator is the go-playground/locales translator for the locale (e.g. en.New()).
	Translator locales.Translator
	// RegisterTranslations wires up the default validator messages for Translator
	// (e.g. the RegisterDefaultTranslations func from a
	// github.com/go-playground/validator/v10/translations/<lang> package).
	RegisterTranslations func(v *govalidator.Validate, trans ut.Translator) error
}

// Config customizes tag name resolution, locale support, and i18n behavior for New.
type Config struct {
	// ContextExtractor resolves the active locale from a context.Context. If unset, ValidateContext
	// always uses DefaultLocale.
	ContextExtractor ContextExtractor
	// CustomLabelTag overrides the struct tag used for the human-readable field name. Defaults to "label".
	CustomLabelTag string
	// CustomJSONTag overrides the struct tag used to resolve the JSON field name. Defaults to "json".
	CustomJSONTag string
	// CustomQueryTag overrides the struct tag used to resolve the query field name. Defaults to "query".
	CustomQueryTag string
	// CustomParamTag overrides the struct tag used to resolve the path param field name. Defaults to "param".
	CustomParamTag string
	// CustomFormTag overrides the struct tag used to resolve the form field name. Defaults to "form".
	CustomFormTag string
	// Locales overrides the set of supported locales. Defaults to English ("en") only.
	Locales []Locale
	// DefaultLocale is used when the context extractor is unset, returns ok=false, or resolves to an
	// unsupported locale. Defaults to the first entry in Locales (or "en" when Locales is unset).
	DefaultLocale string
	// FieldPathFormat controls how FieldError.Field values are rendered. Defaults to FieldPathDot.
	FieldPathFormat FieldPathFormat
}

// defaultLocales returns the built-in English locale used when Config.Locales is unset.
func defaultLocales() []Locale {
	return []Locale{
		{Tag: "en", Translator: en.New(), RegisterTranslations: enTranslations.RegisterDefaultTranslations},
	}
}

// ContextExtractor is a function type that extracts the locale from the context for i18n support.
type ContextExtractor func(ctx context.Context) (locale string, ok bool)

// FieldPathFormat controls how FieldError.Field values are rendered for nested and indexed fields.
type FieldPathFormat int

const (
	// FieldPathDot renders paths as dot notation with bracketed indices, e.g. "phones[0].number".
	// This is the default.
	FieldPathDot FieldPathFormat = iota
	// FieldPathJSONPointer renders paths as RFC 6901 JSON Pointers, e.g. "/phones/0/number".
	FieldPathJSONPointer
)

// New creates a new Validate instance with custom tag name resolution and registered locale translations.
// With no Config, it resolves field names from the "label", "json", "query", "param", and "form" tags
// (in that order) and supports only the English ("en") locale.
func New(cfg ...Config) *Validate {
	v := govalidator.New()

	labelTag := "label"
	jsonTag := "json"
	queryTag := "query"
	paramTag := "param"
	formTag := "form"
	var contextExtractor ContextExtractor
	var customLocales []Locale
	var defaultLocale string
	var fieldPathFormat FieldPathFormat
	if len(cfg) > 0 {
		if cfg[0].CustomLabelTag != "" {
			labelTag = cfg[0].CustomLabelTag
		}
		if cfg[0].CustomJSONTag != "" {
			jsonTag = cfg[0].CustomJSONTag
		}
		if cfg[0].CustomQueryTag != "" {
			queryTag = cfg[0].CustomQueryTag
		}
		if cfg[0].CustomParamTag != "" {
			paramTag = cfg[0].CustomParamTag
		}
		if cfg[0].CustomFormTag != "" {
			formTag = cfg[0].CustomFormTag
		}
		if cfg[0].ContextExtractor != nil {
			contextExtractor = cfg[0].ContextExtractor
		}
		customLocales = cfg[0].Locales
		defaultLocale = cfg[0].DefaultLocale
		fieldPathFormat = cfg[0].FieldPathFormat
	}
	if len(customLocales) == 0 {
		customLocales = defaultLocales()
	}
	if defaultLocale == "" {
		defaultLocale = customLocales[0].Tag
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		if label := fld.Tag.Get(labelTag); label != "" {
			return label
		}
		for _, tag := range []string{jsonTag, queryTag, paramTag, formTag} {
			if name := fld.Tag.Get(tag); name != "" {
				name = strings.Split(name, ",")[0]
				if name != "" && name != "-" {
					return name
				}
			}
		}
		return fld.Name
	})

	translators := make([]locales.Translator, len(customLocales))
	for i, l := range customLocales {
		translators[i] = l.Translator
	}
	uni := ut.New(translators[0], translators...)

	for _, l := range customLocales {
		trans, ok := uni.GetTranslator(l.Tag)
		if !ok || l.RegisterTranslations == nil {
			continue
		}
		_ = l.RegisterTranslations(v, trans)
	}

	return &Validate{
		validate:         v,
		uni:              uni,
		contextExtractor: contextExtractor,
		defaultLocale:    defaultLocale,
		fieldPathFormat:  fieldPathFormat,
		jsonTag:          jsonTag,
		queryTag:         queryTag,
		paramTag:         paramTag,
		formTag:          formTag,
	}
}

// Validate validates the struct using the default context (English locale).
func (v *Validate) Validate(i any) error {
	return v.ValidateContext(context.Background(), i)
}

// getTranslator resolves the translator for the current request locale, falling back to
// v.defaultLocale when the context extractor is unset or its locale isn't registered.
func (v *Validate) getTranslator(ctx context.Context) ut.Translator {
	locale := v.defaultLocale
	if v.contextExtractor != nil {
		if l, ok := v.contextExtractor(ctx); ok {
			locale = l
		}
	}
	trans, ok := v.uni.FindTranslator(locale)
	if !ok {
		trans, _ = v.uni.GetTranslator(v.defaultLocale)
	}
	return trans
}

// ValidateContext validates the struct using the provided context for i18n support.
func (v *Validate) ValidateContext(ctx context.Context, i any) error {
	err := v.validate.StructCtx(ctx, i)
	if err == nil {
		return nil
	}

	var validationErrors govalidator.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok {
		return err
	}

	trans := v.getTranslator(ctx)

	seenFields := make(map[string]struct{}, len(validationErrors))
	fieldErrors := make([]FieldError, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		field := v.jsonFieldPath(i, fieldErr.StructNamespace(), fieldErr.StructField())
		if _, seen := seenFields[field]; seen {
			continue
		}
		seenFields[field] = struct{}{}

		fieldErrors = append(fieldErrors, FieldError{
			Field:   field,
			Message: fieldErr.Translate(trans),
		})
	}

	return NewErrors(fieldErrors...)
}

// pathSegment is one component of a resolved field path: either a field name (e.g. "phones") or
// a slice/array index (e.g. "0"), tracked so dot and JSON Pointer formatting can render it correctly.
type pathSegment struct {
	value   string
	isIndex bool
}

// jsonFieldPath converts a govalidator struct namespace (e.g. "Req.Phones[0].Number") into a path
// built from the configured field-name tags, formatted per v.fieldPathFormat (e.g. "phones[0].number"
// for FieldPathDot or "/phones/0/number" for FieldPathJSONPointer), by walking i's type alongside
// the namespace. It falls back to fallback, formatted the same way, if i's type can't be walked or
// the namespace resolves empty.
func (v *Validate) jsonFieldPath(i any, structNamespace, fallback string) string {
	fallbackSegments := []pathSegment{{value: fallback}}

	if structNamespace == "" {
		return v.formatPath(fallbackSegments)
	}

	t := reflect.TypeOf(i)
	if t == nil {
		return v.formatPath(fallbackSegments)
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return v.formatPath(fallbackSegments)
	}

	parts := strings.Split(structNamespace, ".")
	if len(parts) == 0 {
		return v.formatPath(fallbackSegments)
	}

	// Skip the root struct name in namespace.
	parts = parts[1:]
	if len(parts) == 0 {
		return v.formatPath(fallbackSegments)
	}

	segments := make([]pathSegment, 0, len(parts))
	current := t

	for _, part := range parts {
		name, indexSuffix := splitFieldAndIndex(part)
		if name == "" {
			continue
		}

		jsonName, nextType := v.resolveJSONNameAndType(current, name, indexSuffix)
		segments = append(segments, pathSegment{value: jsonName})
		for _, index := range splitIndices(indexSuffix) {
			segments = append(segments, pathSegment{value: index, isIndex: true})
		}
		current = nextType
	}

	if len(segments) == 0 {
		return v.formatPath(fallbackSegments)
	}

	return v.formatPath(segments)
}

// formatPath renders segments per v.fieldPathFormat.
func (v *Validate) formatPath(segments []pathSegment) string {
	if v.fieldPathFormat == FieldPathJSONPointer {
		var b strings.Builder
		for _, seg := range segments {
			b.WriteByte('/')
			b.WriteString(escapeJSONPointerToken(seg.value))
		}
		return b.String()
	}

	var b strings.Builder
	for i, seg := range segments {
		if seg.isIndex {
			b.WriteByte('[')
			b.WriteString(seg.value)
			b.WriteByte(']')
			continue
		}
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(seg.value)
	}
	return b.String()
}

// escapeJSONPointerToken escapes a path segment per RFC 6901 (~ -> ~0, / -> ~1).
func escapeJSONPointerToken(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// splitIndices splits a bracketed index suffix like "[0][1]" into its individual index
// strings, e.g. ["0", "1"]. It returns nil for an empty suffix.
func splitIndices(indexSuffix string) []string {
	if indexSuffix == "" {
		return nil
	}
	trimmed := strings.Trim(indexSuffix, "[]")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "][")
}

// resolveJSONNameAndType resolves the field-name tag value for the struct field named name on
// current, and returns the (dereferenced, index-unwrapped) type of that field for the next step
// of the namespace walk. It falls back to name itself when current isn't a struct or the field
// isn't found.
func (v *Validate) resolveJSONNameAndType(current reflect.Type, name, indexSuffix string) (string, reflect.Type) {
	current = dereferenceType(current)
	if current.Kind() != reflect.Struct {
		return name, current
	}

	field, ok := current.FieldByName(name)
	if !ok {
		return name, current
	}

	jsonName := ""
	for _, tag := range []string{v.jsonTag, v.queryTag, v.paramTag, v.formTag} {
		if name := field.Tag.Get(tag); name != "" {
			jsonName = strings.Split(name, ",")[0]
			if jsonName != "" && jsonName != "-" {
				break
			}
			jsonName = ""
		}
	}

	if jsonName == "" {
		jsonName = name
	}

	nextType := dereferenceType(field.Type)
	return jsonName, unwrapIndexedType(nextType, indexSuffix)
}

// dereferenceType unwraps successive pointer indirections, returning the underlying element type.
func dereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// unwrapIndexedType unwraps t once per "[" in indexSuffix (e.g. "[0][1]" for a nested slice),
// stepping into each slice/array element type so the namespace walk lands on the element's type
// rather than the collection's.
func unwrapIndexedType(t reflect.Type, indexSuffix string) reflect.Type {
	indexCount := strings.Count(indexSuffix, "[")
	for range indexCount {
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = dereferenceType(t.Elem())
		}
	}
	return t
}

// splitFieldAndIndex splits a govalidator namespace segment like "Phones[0]" into its field
// name ("Phones") and index suffix ("[0]"). Segments without an index are returned unchanged.
func splitFieldAndIndex(part string) (string, string) {
	idx := strings.Index(part, "[")
	if idx == -1 {
		return part, ""
	}
	return part[:idx], part[idx:]
}
