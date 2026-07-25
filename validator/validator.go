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
	headerTag        string
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

// defaultLocales returns the built-in English locale used when no locales are configured via WithLocales.
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

// New creates a new Validate instance with custom tag name resolution and registered locale
// translations, configured via Option values (e.g. WithLocales, WithFieldPathFormat). With no
// options, it resolves field names from the "label", "json", "query", "param", "form", and
// "header" tags (in that order) and supports only the English ("en") locale.
func New(opts ...Option) *Validate {
	o := &options{
		labelTag:  "label",
		jsonTag:   "json",
		queryTag:  "query",
		paramTag:  "param",
		formTag:   "form",
		headerTag: "header",
	}
	for _, opt := range opts {
		opt(o)
	}

	customLocales := o.locales
	if len(customLocales) == 0 {
		customLocales = defaultLocales()
	}
	defaultLocale := o.defaultLocale
	if defaultLocale == "" {
		defaultLocale = customLocales[0].Tag
	}

	v := govalidator.New()

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		if label := fld.Tag.Get(o.labelTag); label != "" {
			return label
		}
		for _, tag := range []string{o.jsonTag, o.queryTag, o.paramTag, o.formTag, o.headerTag} {
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
		contextExtractor: o.contextExtractor,
		defaultLocale:    defaultLocale,
		fieldPathFormat:  o.fieldPathFormat,
		jsonTag:          o.jsonTag,
		queryTag:         o.queryTag,
		paramTag:         o.paramTag,
		formTag:          o.formTag,
		headerTag:        o.headerTag,
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
		field, source := v.jsonFieldPath(i, fieldErr.StructNamespace(), fieldErr.StructField())
		if _, seen := seenFields[field]; seen {
			continue
		}
		seenFields[field] = struct{}{}

		fieldErrors = append(fieldErrors, FieldError{
			Field:   field,
			Message: fieldErr.Translate(trans),
			Source:  source,
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
// the namespace resolves empty. The returned Source reflects which tag (json/query/param/form/header)
// resolved the path's root field, defaulting to SourceBody when none matched.
func (v *Validate) jsonFieldPath(i any, structNamespace, fallback string) (string, Source) {
	fallbackSegments := []pathSegment{{value: fallback}}

	if structNamespace == "" {
		return v.formatPath(fallbackSegments), SourceBody
	}

	t := reflect.TypeOf(i)
	if t == nil {
		return v.formatPath(fallbackSegments), SourceBody
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return v.formatPath(fallbackSegments), SourceBody
	}

	parts := strings.Split(structNamespace, ".")
	if len(parts) == 0 {
		return v.formatPath(fallbackSegments), SourceBody
	}

	// Skip the root struct name in namespace.
	parts = parts[1:]
	if len(parts) == 0 {
		return v.formatPath(fallbackSegments), SourceBody
	}

	segments := make([]pathSegment, 0, len(parts))
	current := t
	source := SourceBody
	sourceResolved := false

	for _, part := range parts {
		name, indexSuffix := splitFieldAndIndex(part)
		if name == "" {
			continue
		}

		jsonName, nextType, segSource := v.resolveJSONNameAndType(current, name, indexSuffix)
		if !sourceResolved {
			source = segSource
			sourceResolved = true
		}
		segments = append(segments, pathSegment{value: jsonName})
		for _, index := range splitIndices(indexSuffix) {
			segments = append(segments, pathSegment{value: index, isIndex: true})
		}
		current = nextType
	}

	if len(segments) == 0 {
		return v.formatPath(fallbackSegments), SourceBody
	}

	return v.formatPath(segments), source
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

// tagSource pairs a configured field-name tag with the Source it represents.
type tagSource struct {
	tag    string
	source Source
}

// tagSources returns the configured field-name tags paired with their Source, in
// resolution priority order.
func (v *Validate) tagSources() []tagSource {
	return []tagSource{
		{v.jsonTag, SourceBody},
		{v.queryTag, SourceQuery},
		{v.paramTag, SourceParam},
		{v.formTag, SourceForm},
		{v.headerTag, SourceHeader},
	}
}

// resolveJSONNameAndType resolves the field-name tag value for the struct field named name on
// current, and returns the (dereferenced, index-unwrapped) type of that field for the next step
// of the namespace walk, along with the Source of the tag that resolved the name (SourceBody if
// none matched). It falls back to name itself when current isn't a struct or the field isn't found.
func (v *Validate) resolveJSONNameAndType(current reflect.Type, name, indexSuffix string) (string, reflect.Type, Source) {
	current = dereferenceType(current)
	if current.Kind() != reflect.Struct {
		return name, current, SourceBody
	}

	field, ok := current.FieldByName(name)
	if !ok {
		return name, current, SourceBody
	}

	jsonName := ""
	source := SourceBody
	for _, ts := range v.tagSources() {
		if tagVal := field.Tag.Get(ts.tag); tagVal != "" {
			jsonName = strings.Split(tagVal, ",")[0]
			if jsonName != "" && jsonName != "-" {
				source = ts.source
				break
			}
			jsonName = ""
		}
	}

	if jsonName == "" {
		jsonName = name
	}

	nextType := dereferenceType(field.Type)
	return jsonName, unwrapIndexedType(nextType, indexSuffix), source
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
