package validator

// Option customizes tag name resolution, locale support, and i18n behavior for New.
type Option func(*options)

// options holds the settings assembled from the Option values passed to New.
type options struct {
	contextExtractor ContextExtractor
	labelTag         string
	jsonTag          string
	queryTag         string
	paramTag         string
	formTag          string
	headerTag        string
	locales          []Locale
	defaultLocale    string
	fieldPathFormat  FieldPathFormat
}

// WithContextExtractor sets the function ValidateContext uses to resolve the active locale
// from a context.Context. Without this option, ValidateContext always uses the default locale.
func WithContextExtractor(fn ContextExtractor) Option {
	return func(o *options) { o.contextExtractor = fn }
}

// WithLabelTag overrides the struct tag used for the human-readable field name. Defaults to
// "label". A label always wins over the json/query/param/form/header tag chain, but an
// AttributesProvider entry for the field still wins over the label.
func WithLabelTag(tag string) Option {
	return func(o *options) { o.labelTag = tag }
}

// WithJSONTag overrides the struct tag used to resolve the JSON field name. Defaults to "json".
func WithJSONTag(tag string) Option {
	return func(o *options) { o.jsonTag = tag }
}

// WithQueryTag overrides the struct tag used to resolve the query field name. Defaults to "query".
func WithQueryTag(tag string) Option {
	return func(o *options) { o.queryTag = tag }
}

// WithParamTag overrides the struct tag used to resolve the path param field name. Defaults to "param".
func WithParamTag(tag string) Option {
	return func(o *options) { o.paramTag = tag }
}

// WithFormTag overrides the struct tag used to resolve the form field name. Defaults to "form".
func WithFormTag(tag string) Option {
	return func(o *options) { o.formTag = tag }
}

// WithHeaderTag overrides the struct tag used to resolve the header field name. Defaults to "header".
func WithHeaderTag(tag string) Option {
	return func(o *options) { o.headerTag = tag }
}

// WithLocales overrides the set of supported locales. Without this option, only English
// ("en") is registered.
func WithLocales(locales ...Locale) Option {
	return func(o *options) { o.locales = locales }
}

// WithDefaultLocale sets the locale used when the context extractor is unset, returns ok=false,
// or resolves to an unsupported locale. Without this option, it defaults to the first locale
// passed to WithLocales (or "en" when WithLocales is unset).
func WithDefaultLocale(locale string) Option {
	return func(o *options) { o.defaultLocale = locale }
}

// WithFieldPathFormat controls how FieldError.Field values are rendered. Defaults to FieldPathDot.
func WithFieldPathFormat(format FieldPathFormat) Option {
	return func(o *options) { o.fieldPathFormat = format }
}
