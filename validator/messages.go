package validator

// MessagesProvider lets a validated value supply custom validation messages, overriding the
// translated default for a specific field/rule combination. Keys are "<field>.<rule>", built
// from the resolved field path (dot notation, "*" in place of any slice/array index) and the
// validator rule name — e.g. "title.required" or "phones.*.number.required".
type MessagesProvider interface {
	Messages() map[string]string
}

// AttributesProvider lets a validated value supply custom human-readable field names, used in
// place of the tag-resolved name when rendering translated messages. Keys use the same resolved
// field path as MessagesProvider, without the rule suffix — e.g. "email" or "phones.*.number".
type AttributesProvider interface {
	Attributes() map[string]string
}
