package validator

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"

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
	labelTag         string
	jsonTag          string
	queryTag         string
	paramTag         string
	formTag          string
	headerTag        string
	// tagSources is precomputed once in New, in resolution priority order, so resolveTagName
	// doesn't rebuild it on every field it resolves.
	tagSources []tagSource
	// pathCache memoizes jsonFieldPath's result per (struct type, govalidator namespace): both
	// are fixed for a given struct schema, so repeat validation failures on the same field (the
	// common case for a long-lived, shared Validate) skip the reflect walk entirely. Safe for
	// concurrent use; never invalidated since entries are pure functions of the key.
	pathCache sync.Map // map[pathCacheKey]cachedPath
}

// pathCacheKey identifies one jsonFieldPath resolution: a struct type plus the govalidator
// namespace within it (e.g. "Req.Phones[0].Number").
type pathCacheKey struct {
	typ       reflect.Type
	namespace string
}

// cachedPath is jsonFieldPath's resolved result for a pathCacheKey.
type cachedPath struct {
	field      string
	source     Source
	keyPath    string
	parentType reflect.Type
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
// options, it resolves field names from the "label" tag, then the "json", "query", "param",
// "form", and "header" tags (in that order), and supports only the English ("en") locale.
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
		labelTag:         o.labelTag,
		jsonTag:          o.jsonTag,
		queryTag:         o.queryTag,
		paramTag:         o.paramTag,
		formTag:          o.formTag,
		headerTag:        o.headerTag,
		tagSources: []tagSource{
			{o.jsonTag, SourceBody},
			{o.queryTag, SourceQuery},
			{o.paramTag, SourceParam},
			{o.formTag, SourceForm},
			{o.headerTag, SourceHeader},
		},
	}
}

// Validate validates the struct using the default context (English locale).
func (v *Validate) Validate(i any) error {
	return v.ValidateContext(context.Background(), i)
}

// RegisterValidation registers a custom validation function for tag, so struct fields tagged
// validate:"<tag>" are checked by fn. It wraps govalidator.Validate.RegisterValidation directly;
// callValidationEvenIfNull follows the same semantics as the underlying library (fn runs even
// when the field is a nil pointer or interface).
func (v *Validate) RegisterValidation(tag string, fn govalidator.Func, callValidationEvenIfNull ...bool) error {
	return v.validate.RegisterValidation(tag, fn, callValidationEvenIfNull...)
}

// RegisterValidationCtx is the context-aware variant of RegisterValidation, for validation
// functions that need access to the context.Context passed to ValidateContext.
func (v *Validate) RegisterValidationCtx(tag string, fn govalidator.FuncCtx, callValidationEvenIfNull ...bool) error {
	return v.validate.RegisterValidationCtx(tag, fn, callValidationEvenIfNull...)
}

// RegisterCustomTypeFunc registers fn to extract a comparable value from any of types before
// validation rules run against it. It wraps govalidator.Validate.RegisterCustomTypeFunc directly.
func (v *Validate) RegisterCustomTypeFunc(fn govalidator.CustomTypeFunc, types ...any) {
	v.validate.RegisterCustomTypeFunc(fn, types...)
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

	var msgs, attrs map[string]string
	if mp, ok := i.(MessagesProvider); ok {
		msgs = mp.Messages()
	}
	if ap, ok := i.(AttributesProvider); ok {
		attrs = ap.Attributes()
	}

	seenFields := make(map[string]struct{}, len(validationErrors))
	fieldErrors := make([]FieldError, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		field, source, keyPath, parentType := v.jsonFieldPath(i, fieldErr.StructNamespace(), fieldErr.StructField())
		if _, seen := seenFields[field]; seen {
			continue
		}
		seenFields[field] = struct{}{}

		message := fieldErr.Translate(trans)
		if custom, ok := msgs[keyPath+"."+fieldErr.Tag()]; ok {
			message = custom
		} else {
			if attr, ok := attrs[keyPath]; ok {
				if leaf := fieldErr.Field(); leaf != "" {
					message = strings.Replace(message, leaf, attr, 1)
				}
			}
			// Cross-field rules (eqfield, gtfield, ...) embed the compared field's raw struct
			// field name via Param(), untouched by RegisterTagNameFunc; resolve and substitute
			// it the same way so AttributesProvider can override it too.
			if param := fieldErr.Param(); param != "" && parentType != nil {
				if field, ok := dereferenceType(parentType).FieldByName(param); ok {
					paramName, _ := v.resolveTagName(field, param)
					replacement := paramName
					if label := field.Tag.Get(v.labelTag); label != "" {
						replacement = label
					}
					if attr, ok := attrs[siblingMessageKey(keyPath, paramName)]; ok {
						replacement = attr
					}
					message = strings.Replace(message, param, replacement, 1)
				}
			}
		}

		fieldErrors = append(fieldErrors, FieldError{
			Field:   field,
			Message: message,
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
// resolved the path's root field, defaulting to SourceBody when none matched. The third result is a
// message key: the same path in dot notation with "*" for every index, used to look up
// MessagesProvider/AttributesProvider overrides independent of v.fieldPathFormat. The fourth
// result is the reflect.Type of the struct that directly declares the resolved field — used to
// resolve sibling field names (e.g. govalidator.FieldError.Param() for eqfield/gtfield/etc.) for
// AttributesProvider substitution. It is nil when the path couldn't be walked.
func (v *Validate) jsonFieldPath(i any, structNamespace, fallback string) (string, Source, string, reflect.Type) {
	fallbackSegments := []pathSegment{{value: fallback}}

	if structNamespace == "" {
		return v.formatPath(fallbackSegments), SourceBody, formatMessageKey(fallbackSegments), nil
	}

	t := reflect.TypeOf(i)
	if t == nil {
		return v.formatPath(fallbackSegments), SourceBody, formatMessageKey(fallbackSegments), nil
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return v.formatPath(fallbackSegments), SourceBody, formatMessageKey(fallbackSegments), nil
	}

	cacheKey := pathCacheKey{typ: t, namespace: structNamespace}
	if cached, ok := v.pathCache.Load(cacheKey); ok {
		cp := cached.(cachedPath)
		return cp.field, cp.source, cp.keyPath, cp.parentType
	}

	parts := strings.Split(structNamespace, ".")
	if len(parts) == 0 {
		return v.formatPath(fallbackSegments), SourceBody, formatMessageKey(fallbackSegments), nil
	}

	// Skip the root struct name in namespace.
	parts = parts[1:]
	if len(parts) == 0 {
		return v.formatPath(fallbackSegments), SourceBody, formatMessageKey(fallbackSegments), nil
	}

	segments := make([]pathSegment, 0, len(parts))
	current := t
	parentType := t
	source := SourceBody
	sourceResolved := false

	for _, part := range parts {
		name, indexSuffix := splitFieldAndIndex(part)
		if name == "" {
			continue
		}

		parentType = current
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
		return v.formatPath(fallbackSegments), SourceBody, formatMessageKey(fallbackSegments), nil
	}

	field, keyPath := v.formatPath(segments), formatMessageKey(segments)
	v.pathCache.Store(cacheKey, cachedPath{field: field, source: source, keyPath: keyPath, parentType: parentType})
	return field, source, keyPath, parentType
}

// siblingMessageKey builds the message key for a field named name declared in the same struct as
// the field keyed by keyPath — i.e. keyPath with its last dot-segment replaced by name. Used to
// look up AttributesProvider overrides for sibling fields referenced by cross-field rules.
func siblingMessageKey(keyPath, name string) string {
	if idx := strings.LastIndex(keyPath, "."); idx >= 0 {
		return keyPath[:idx+1] + name
	}
	return name
}

// formatMessageKey renders segments as a dot path with "*" standing in for slice/array indices,
// independent of v.fieldPathFormat, used to key MessagesProvider and AttributesProvider maps.
func formatMessageKey(segments []pathSegment) string {
	var b strings.Builder
	for i, seg := range segments {
		if seg.isIndex {
			b.WriteString(".*")
			continue
		}
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(seg.value)
	}
	return b.String()
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

// resolveTagName resolves field's json/query/param/form/header-tag name, falling back to name,
// along with the Source of whichever tag matched (SourceBody if none did).
func (v *Validate) resolveTagName(field reflect.StructField, name string) (string, Source) {
	for _, ts := range v.tagSources {
		if tagVal := field.Tag.Get(ts.tag); tagVal != "" {
			jsonName, _, _ := strings.Cut(tagVal, ",")
			if jsonName != "" && jsonName != "-" {
				return jsonName, ts.source
			}
		}
	}
	return name, SourceBody
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

	jsonName, source := v.resolveTagName(field, name)
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
