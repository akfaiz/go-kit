package problem_test

import (
	"errors"
	"testing"

	"github.com/akfaiz/go-kit/problem"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := problem.New("Not Found", "not_found", 404)

	assert.Equal(t, "not_found", err.Type)
	assert.Equal(t, "Not Found", err.Title)
	assert.Equal(t, 404, err.Status)
	assert.Empty(t, err.Detail)
}

func TestNew_WithDetailArg(t *testing.T) {
	err := problem.New("Not Found", "not_found", 404, "resource missing")

	assert.Equal(t, "resource missing", err.Detail)
}

func TestRegister(t *testing.T) {
	errFn := problem.Register("Not Found", "not_found", 404)

	err := errFn()

	assert.Equal(t, "not_found", err.Type)
	assert.Equal(t, "Not Found", err.Title)
	assert.Equal(t, 404, err.Status)
	assert.Empty(t, err.Detail)
}

func TestRegister_UsesDefaultDetailWhenNotOverridden(t *testing.T) {
	errFn := problem.Register("Not Found", "not_found", 404, "default detail")

	err := errFn()

	assert.Equal(t, "default detail", err.Detail)
}

func TestRegister_OverridesDefaultDetail(t *testing.T) {
	errFn := problem.Register("Not Found", "not_found", 404, "default detail")

	err := errFn("overridden detail")

	assert.Equal(t, "overridden detail", err.Detail)
}

func TestError_Error(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := problem.New("Not Found", "not_found", 404, "resource missing")
		assert.Equal(t, "Not Found: resource missing", err.Error())
	})

	t.Run("without detail", func(t *testing.T) {
		err := problem.New("Not Found", "not_found", 404)
		assert.Equal(t, "Not Found", err.Error())
	})
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("boom")
	err := problem.New("Internal Server Error", "internal_error", 500).
		WithCause(cause)

	assert.True(t, errors.Is(err, cause))
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestError_Is(t *testing.T) {
	errNotFound := problem.Register("Not Found", "not_found", 404)

	t.Run("matches by type across distinct instances", func(t *testing.T) {
		err := errNotFound("user missing")

		assert.True(t, errors.Is(err, errNotFound()))
	})

	t.Run("does not match different type", func(t *testing.T) {
		errBadRequest := problem.Register("Bad Request", "bad_request", 400)

		assert.False(t, errors.Is(errNotFound(), errBadRequest()))
	})

	t.Run("does not match non-Error target", func(t *testing.T) {
		assert.False(t, errors.Is(errNotFound(), errors.New("boom")))
	})
}

func TestError_WithDetail(t *testing.T) {
	err := problem.New("Bad Request", "bad_request", 400)

	result := err.WithDetail("email is invalid")

	assert.Same(t, err, result)
	assert.Equal(t, "email is invalid", err.Detail)
}

func TestError_WithErrors(t *testing.T) {
	err := problem.New("Bad Request", "bad_request", 400)
	fieldErrors := map[string]string{"email": "must be a valid email"}

	result := err.WithErrors(fieldErrors)

	assert.Same(t, err, result)
	assert.Equal(t, fieldErrors, err.Errors)
}

func TestError_WithInstance(t *testing.T) {
	err := problem.New("Not Found", "not_found", 404)

	result := err.WithInstance("/users/123")

	assert.Same(t, err, result)
	assert.Equal(t, "/users/123", err.Instance)
}
