package validator_test

import (
	"context"
	"testing"

	"github.com/akfaiz/go-kit/validator"
	"github.com/go-playground/locales/fr"
	frTranslations "github.com/go-playground/validator/v10/translations/fr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registerRequest struct {
	Email                string `json:"email" validate:"required,email" label:"Email"`
	Password             string `json:"password" validate:"required,min=8" label:"Password"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password" label:"Confirm Password"`
}

type nestedProfileRequest struct {
	Profile nestedProfile `json:"profile"`
}

type nestedProfile struct {
	Street string `json:"street" validate:"required" label:"Street"`
}

type stringArrayRequest struct {
	Tags []string `json:"tags" validate:"required,dive,required" label:"Tags"`
}

type arrayStructRequest struct {
	Phones []phone `json:"phones" validate:"required,dive" label:"Phones"`
}

type phone struct {
	Number string `json:"number" validate:"required" label:"Number"`
}

type queryRequest struct {
	Page int `query:"page" validate:"required,min=1" label:"Page"`
}

type paramRequest struct {
	ID string `param:"id" validate:"required" label:"ID"`
}

type formRequest struct {
	Name string `form:"name" validate:"required" label:"Name"`
}

type priorityRequest struct {
	Field string `json:"json_field" query:"query_field" validate:"required" label:"Label Field"`
}

func TestValidate_Success(t *testing.T) {
	v := validator.New()
	req := &registerRequest{
		Email:                "john.doe@example.com",
		Password:             "supersecret",
		PasswordConfirmation: "supersecret",
	}

	err := v.Validate(req)
	require.NoError(t, err)
}

func TestValidate_ReturnsValidationErrorWithJSONFieldAndTranslatedMessage(t *testing.T) {
	v := validator.New()
	req := &registerRequest{
		Email:                "string",
		Password:             "supersecret",
		PasswordConfirmation: "supersecret",
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)
	require.Len(t, *vErr, 1)

	first := vErr.First()
	require.NotNil(t, first)
	assert.Equal(t, "email", first.Field)
	assert.Equal(t, "Email must be a valid email address", first.Message)
}

func TestValidate_UsesLabelInMessageAndJSONFieldKey(t *testing.T) {
	v := validator.New()
	req := &registerRequest{
		Email:                "john.doe@example.com",
		Password:             "",
		PasswordConfirmation: "",
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)

	found := map[string]string{}
	for _, fieldErr := range *vErr {
		found[fieldErr.Field] = fieldErr.Message
	}

	assert.Equal(t, "Password is a required field", found["password"])
	assert.Equal(t, "Confirm Password is a required field", found["password_confirmation"])
}

func TestValidate_NestedStructPathUsesJSONKeys(t *testing.T) {
	v := validator.New()
	req := &nestedProfileRequest{
		Profile: nestedProfile{
			Street: "",
		},
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)
	require.NotNil(t, vErr.First())

	assert.Equal(t, "profile.street", vErr.First().Field)
	assert.Equal(t, "Street is a required field", vErr.First().Message)
}

func TestValidate_ArrayPathUsesJSONKeysWithIndex(t *testing.T) {
	v := validator.New()
	req := &stringArrayRequest{
		Tags: []string{""},
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)
	require.NotNil(t, vErr.First())

	assert.Equal(t, "tags[0]", vErr.First().Field)
	assert.Equal(t, "Tags[0] is a required field", vErr.First().Message)
}

func TestValidate_ArrayStructPathUsesJSONKeysWithIndex(t *testing.T) {
	v := validator.New()
	req := &arrayStructRequest{
		Phones: []phone{
			{Number: ""},
		},
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)
	require.NotNil(t, vErr.First())

	assert.Equal(t, "phones[0].number", vErr.First().Field)
	assert.Equal(t, "Number is a required field", vErr.First().Message)
}

func TestValidate_ArrayStructPathReportsErrorPerElement(t *testing.T) {
	v := validator.New()
	req := &arrayStructRequest{
		Phones: []phone{
			{Number: ""},
			{Number: ""},
		},
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)
	require.Len(t, *vErr, 2)

	assert.Equal(t, "phones[0].number", (*vErr)[0].Field)
	assert.Equal(t, "phones[1].number", (*vErr)[1].Field)
}

type localeKeyType struct{}

var localeKey = localeKeyType{}

func TestValidate_JSONPointerFieldPathFormat(t *testing.T) {
	v := validator.New(validator.Config{
		FieldPathFormat: validator.FieldPathJSONPointer,
	})

	t.Run("nested struct", func(t *testing.T) {
		req := &nestedProfileRequest{Profile: nestedProfile{Street: ""}}

		err := v.Validate(req)
		require.Error(t, err)

		var vErr *validator.ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "/profile/street", vErr.First().Field)
	})

	t.Run("array of scalars", func(t *testing.T) {
		req := &stringArrayRequest{Tags: []string{""}}

		err := v.Validate(req)
		require.Error(t, err)

		var vErr *validator.ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "/tags/0", vErr.First().Field)
	})

	t.Run("array of structs", func(t *testing.T) {
		req := &arrayStructRequest{Phones: []phone{{Number: ""}, {Number: ""}}}

		err := v.Validate(req)
		require.Error(t, err)

		var vErr *validator.ValidationError
		require.ErrorAs(t, err, &vErr)
		require.Len(t, *vErr, 2)
		assert.Equal(t, "/phones/0/number", (*vErr)[0].Field)
		assert.Equal(t, "/phones/1/number", (*vErr)[1].Field)
	})

	t.Run("top-level field", func(t *testing.T) {
		req := &formRequest{Name: ""}

		err := v.Validate(req)
		require.Error(t, err)

		var vErr *validator.ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "/name", vErr.First().Field)
	})
}

func TestValidate_SupportsCustomLocales(t *testing.T) {
	v := validator.New(validator.Config{
		Locales: []validator.Locale{
			{Tag: "fr", Translator: fr.New(), RegisterTranslations: frTranslations.RegisterDefaultTranslations},
		},
		ContextExtractor: func(ctx context.Context) (string, bool) {
			locale, ok := ctx.Value(localeKey).(string)
			return locale, ok
		},
	})
	req := &formRequest{Name: ""}

	ctx := context.WithValue(context.Background(), localeKey, "fr")
	err := v.ValidateContext(ctx, req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "name", vErr.First().Field)
	assert.Equal(t, "Name est un champ obligatoire", vErr.First().Message)
}

func TestValidate_UnsupportedLocaleFallsBackToDefault(t *testing.T) {
	v := validator.New(validator.Config{
		ContextExtractor: func(ctx context.Context) (string, bool) {
			return "zz", true
		},
	})
	req := &formRequest{Name: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "Name is a required field", vErr.First().Message)
}

func TestValidate_SupportsQueryTags(t *testing.T) {
	v := validator.New()
	req := &queryRequest{Page: 0}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "page", vErr.First().Field)
	assert.Equal(t, "Page is a required field", vErr.First().Message)
}

func TestValidate_SupportsParamTags(t *testing.T) {
	v := validator.New()
	req := &paramRequest{ID: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "id", vErr.First().Field)
	assert.Equal(t, "ID is a required field", vErr.First().Message)
}

func TestValidate_SupportsFormTags(t *testing.T) {
	v := validator.New()
	req := &formRequest{Name: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "name", vErr.First().Field)
	assert.Equal(t, "Name is a required field", vErr.First().Message)
}

func TestValidate_TagPriority(t *testing.T) {
	v := validator.New()
	req := &priorityRequest{Field: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "json_field", vErr.First().Field)
}
