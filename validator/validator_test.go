package validator_test

import (
	"context"
	"testing"

	"github.com/akfaiz/go-kit/validator"
	"github.com/go-playground/locales/fr"
	govalidator "github.com/go-playground/validator/v10"
	frTranslations "github.com/go-playground/validator/v10/translations/fr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registerRequest struct {
	Email                string `json:"email" validate:"required,email"`
	Password             string `json:"password" validate:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
}

type nestedProfileRequest struct {
	Profile nestedProfile `json:"profile"`
}

type nestedProfile struct {
	Street string `json:"street" validate:"required"`
}

type stringArrayRequest struct {
	Tags []string `json:"tags" validate:"required,dive,required"`
}

type arrayStructRequest struct {
	Phones []phone `json:"phones" validate:"required,dive"`
}

type phone struct {
	Number string `json:"number" validate:"required"`
}

type queryRequest struct {
	Page int `query:"page" validate:"required,min=1"`
}

type paramRequest struct {
	ID string `param:"id" validate:"required"`
}

type formRequest struct {
	Name string `form:"name" validate:"required"`
}

type headerRequest struct {
	Authorization string `header:"Authorization" validate:"required"`
}

type mixedSourceRequest struct {
	Page          int           `query:"page" validate:"required,min=1"`
	ID            string        `param:"id" validate:"required"`
	Authorization string        `header:"Authorization" validate:"required"`
	Profile       nestedProfile `json:"profile"`
}

type priorityRequest struct {
	Field string `json:"json_field" query:"query_field" validate:"required"`
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
	assert.Equal(t, "email must be a valid email address", first.Message)
}

type registerRequestWithAttributes struct {
	Email                string `json:"email" validate:"required,email"`
	Password             string `json:"password" validate:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
}

func (registerRequestWithAttributes) Attributes() map[string]string {
	return map[string]string{
		"password":              "Password",
		"password_confirmation": "Confirm Password",
	}
}

func TestValidate_AttributesProviderOverridesFieldNameInMessage(t *testing.T) {
	v := validator.New()
	req := &registerRequestWithAttributes{
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

type createPostRequest struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"required"`
}

func (createPostRequest) Messages() map[string]string {
	return map[string]string{
		"title.required": "A title is required",
		"body.required":  "A message is required",
	}
}

func TestValidate_MessagesProviderOverridesTranslatedMessage(t *testing.T) {
	v := validator.New()
	req := &createPostRequest{}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr, "expected *ValidationError, got %T", err)

	found := map[string]string{}
	for _, fieldErr := range *vErr {
		found[fieldErr.Field] = fieldErr.Message
	}

	assert.Equal(t, "A title is required", found["title"])
	assert.Equal(t, "A message is required", found["body"])
}

type arrayStructRequestWithMessages struct {
	Phones []phone `json:"phones" validate:"required,dive"`
}

func (arrayStructRequestWithMessages) Messages() map[string]string {
	return map[string]string{
		"phones.*.number.required": "Every phone needs a number",
	}
}

func TestValidate_MessagesProviderSupportsWildcardArrayKey(t *testing.T) {
	v := validator.New()
	req := &arrayStructRequestWithMessages{
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

	assert.Equal(t, "Every phone needs a number", (*vErr)[0].Message)
	assert.Equal(t, "Every phone needs a number", (*vErr)[1].Message)
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
	assert.Equal(t, "street is a required field", vErr.First().Message)
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
	assert.Equal(t, "tags[0] is a required field", vErr.First().Message)
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
	assert.Equal(t, "number is a required field", vErr.First().Message)
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
	v := validator.New(validator.WithFieldPathFormat(validator.FieldPathJSONPointer))

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
	v := validator.New(
		validator.WithLocales(
			validator.Locale{Tag: "fr", Translator: fr.New(), RegisterTranslations: frTranslations.RegisterDefaultTranslations},
		),
		validator.WithContextExtractor(func(ctx context.Context) (string, bool) {
			locale, ok := ctx.Value(localeKey).(string)
			return locale, ok
		}),
	)
	req := &formRequest{Name: ""}

	ctx := context.WithValue(context.Background(), localeKey, "fr")
	err := v.ValidateContext(ctx, req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "name", vErr.First().Field)
	assert.Equal(t, "name est un champ obligatoire", vErr.First().Message)
}

func TestValidate_UnsupportedLocaleFallsBackToDefault(t *testing.T) {
	v := validator.New(validator.WithContextExtractor(func(ctx context.Context) (string, bool) {
		return "zz", true
	}))
	req := &formRequest{Name: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "name is a required field", vErr.First().Message)
}

func TestValidate_SupportsQueryTags(t *testing.T) {
	v := validator.New()
	req := &queryRequest{Page: 0}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "page", vErr.First().Field)
	assert.Equal(t, "page is a required field", vErr.First().Message)
}

func TestValidate_SupportsParamTags(t *testing.T) {
	v := validator.New()
	req := &paramRequest{ID: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "id", vErr.First().Field)
	assert.Equal(t, "id is a required field", vErr.First().Message)
}

func TestValidate_SupportsFormTags(t *testing.T) {
	v := validator.New()
	req := &formRequest{Name: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "name", vErr.First().Field)
	assert.Equal(t, "name is a required field", vErr.First().Message)
}

func TestValidate_SupportsHeaderTags(t *testing.T) {
	v := validator.New()
	req := &headerRequest{Authorization: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "Authorization", vErr.First().Field)
	assert.Equal(t, "Authorization is a required field", vErr.First().Message)
	assert.Equal(t, validator.SourceHeader, vErr.First().Source)
}

func TestValidate_FieldErrorSourcePerTag(t *testing.T) {
	v := validator.New()
	req := &mixedSourceRequest{
		Page:          0,
		ID:            "",
		Authorization: "",
		Profile:       nestedProfile{Street: ""},
	}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)

	sources := map[string]validator.Source{}
	for _, fieldErr := range *vErr {
		sources[fieldErr.Field] = fieldErr.Source
	}

	assert.Equal(t, validator.SourceQuery, sources["page"])
	assert.Equal(t, validator.SourceParam, sources["id"])
	assert.Equal(t, validator.SourceHeader, sources["Authorization"])
	assert.Equal(t, validator.SourceBody, sources["profile.street"])
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

type customTagRequest struct {
	Name string `custom_json:"full_name" validate:"required"`
}

func (customTagRequest) Attributes() map[string]string {
	return map[string]string{"full_name": "Full Name"}
}

func TestValidate_CustomTags(t *testing.T) {
	v := validator.New(validator.WithJSONTag("custom_json"))
	req := &customTagRequest{Name: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "full_name", vErr.First().Field)
	assert.Equal(t, "Full Name is a required field", vErr.First().Message)
}

func TestValidate_RegisterValidationAddsCustomRule(t *testing.T) {
	type usernameRequest struct {
		Username string `json:"username" validate:"required,notreserved"`
	}

	v := validator.New()
	err := v.RegisterValidation("notreserved", func(fl govalidator.FieldLevel) bool {
		return fl.Field().String() != "admin"
	})
	require.NoError(t, err)

	req := &usernameRequest{Username: "admin"}
	valErr := v.Validate(req)
	require.Error(t, valErr)

	var vErr *validator.ValidationError
	require.ErrorAs(t, valErr, &vErr)
	assert.Equal(t, "username", vErr.First().Field)

	req.Username = "someone-else"
	assert.NoError(t, v.Validate(req))
}

func TestValidate_RegisterValidationCtxReceivesContext(t *testing.T) {
	type ctxKeyType struct{}
	ctxKey := ctxKeyType{}

	type widgetRequest struct {
		Name string `json:"name" validate:"required,matchescontext"`
	}

	v := validator.New()
	err := v.RegisterValidationCtx("matchescontext", func(ctx context.Context, fl govalidator.FieldLevel) bool {
		expected, _ := ctx.Value(ctxKey).(string)
		return fl.Field().String() == expected
	})
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), ctxKey, "widget")
	require.NoError(t, v.ValidateContext(ctx, &widgetRequest{Name: "widget"}))

	valErr := v.ValidateContext(ctx, &widgetRequest{Name: "gadget"})
	require.Error(t, valErr)

	var vErr *validator.ValidationError
	require.ErrorAs(t, valErr, &vErr)
	assert.Equal(t, "name", vErr.First().Field)
}

func TestValidate_CustomHeaderTag(t *testing.T) {
	type customHeaderRequest struct {
		Auth string `custom_header:"Authorization" validate:"required"`
	}

	v := validator.New(validator.WithHeaderTag("custom_header"))
	req := &customHeaderRequest{Auth: ""}

	err := v.Validate(req)
	require.Error(t, err)

	var vErr *validator.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "Authorization", vErr.First().Field)
	assert.Equal(t, validator.SourceHeader, vErr.First().Source)
}
