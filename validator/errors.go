package validator

import (
	"fmt"
	"strings"
)

// FieldError represents a validation error for a specific field, containing the field name and the error message.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError represents a collection of field errors that occurred during validation.
type ValidationError []FieldError

// NewError creates a new ValidationError with a single field error, given the field name and the error message.
func NewError(field, message string) *ValidationError {
	return &ValidationError{{Field: field, Message: message}}
}

// NewErrors creates a new ValidationError with multiple field errors, given a variadic list of FieldError instances.
func NewErrors(fieldErrors ...FieldError) *ValidationError {
	validationErr := ValidationError(fieldErrors)
	return &validationErr
}

// Error implements the error interface, joining all field messages into a single summary string.
func (ve ValidationError) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Message)
	}
	return fmt.Sprintf("validation error: %s", strings.Join(messages, ", "))
}

// Errors returns the underlying slice of field errors.
func (ve ValidationError) Errors() []FieldError {
	return ve
}

// First returns the first field error, or nil if ve is empty.
func (ve ValidationError) First() *FieldError {
	if len(ve) == 0 {
		return nil
	}
	return &ve[0]
}

// Fields returns the field name of every field error, in order.
func (ve *ValidationError) Fields() []string {
	fields := make([]string, len(*ve))
	for i, err := range *ve {
		fields[i] = err.Field
	}
	return fields
}

// Messages returns the message of every field error, in order.
func (ve *ValidationError) Messages() []string {
	messages := make([]string, len(*ve))
	for i, err := range *ve {
		messages[i] = err.Message
	}
	return messages
}

// Add appends a new field error to the ValidationError with the specified field name and error message, and returns the updated ValidationError for chaining.
func (ve *ValidationError) Add(field, message string) *ValidationError {
	*ve = append(*ve, FieldError{Field: field, Message: message})
	return ve
}

// Addf appends a new field error to the ValidationError with the specified field name and a formatted error message, and returns the updated ValidationError for chaining.
func (ve *ValidationError) Addf(field, format string, args ...any) *ValidationError {
	message := fmt.Sprintf(format, args...)
	*ve = append(*ve, FieldError{Field: field, Message: message})
	return ve
}
