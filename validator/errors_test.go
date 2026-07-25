package validator_test

import (
	"testing"

	"github.com/akfaiz/go-kit/validator"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	err := validator.NewError("field", "message")
	assert.Len(t, *err, 1)
	assert.Equal(t, "field", (*err)[0].Field)
	assert.Equal(t, "message", (*err)[0].Message)
}

func TestValidationError_Error(t *testing.T) {
	ve := validator.ValidationError{
		{Field: "f1", Message: "m1"},
		{Field: "f2", Message: "m2"},
	}
	assert.Equal(t, "validation error: m1, m2", ve.Error())
}

func TestValidationError_Errors(t *testing.T) {
	ve := validator.ValidationError{
		{Field: "f1", Message: "m1"},
	}
	assert.Equal(t, []validator.FieldError(ve), ve.Errors())
}

func TestValidationError_First(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var ve validator.ValidationError
		assert.Nil(t, ve.First())
	})
	t.Run("not empty", func(t *testing.T) {
		ve := validator.ValidationError{{Field: "f1", Message: "m1"}}
		assert.Equal(t, &ve[0], ve.First())
	})
}

func TestValidationError_Fields(t *testing.T) {
	ve := validator.NewErrors(
		validator.FieldError{Field: "f1", Message: "m1"},
		validator.FieldError{Field: "f2", Message: "m2"},
	)
	assert.Equal(t, []string{"f1", "f2"}, ve.Fields())
}

func TestValidationError_Messages(t *testing.T) {
	ve := validator.NewErrors(
		validator.FieldError{Field: "f1", Message: "m1"},
		validator.FieldError{Field: "f2", Message: "m2"},
	)
	assert.Equal(t, []string{"m1", "m2"}, ve.Messages())
}

func TestValidationError_Add(t *testing.T) {
	ve := validator.NewErrors()
	ve.Add("f1", "m1")
	assert.Len(t, *ve, 1)
	assert.Equal(t, "f1", (*ve)[0].Field)
}

func TestValidationError_Addf(t *testing.T) {
	ve := validator.NewErrors()
	ve.Addf("f1", "hello %s", "world")
	assert.Len(t, *ve, 1)
	assert.Equal(t, "hello world", (*ve)[0].Message)
}
