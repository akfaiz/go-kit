package validator_test

import (
	"testing"

	"github.com/akfaiz/go-kit/validator"
)

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		validator.New()
	}
}

func BenchmarkValidate_Success(b *testing.B) {
	v := validator.New()
	req := &registerRequest{
		Email:                "john.doe@example.com",
		Password:             "supersecret",
		PasswordConfirmation: "supersecret",
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidate_SingleFieldError(b *testing.B) {
	v := validator.New()
	req := &registerRequest{
		Email:                "not-an-email",
		Password:             "supersecret",
		PasswordConfirmation: "supersecret",
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(req); err == nil {
			b.Fatal("expected error")
		}
	}
}

func BenchmarkValidate_CrossFieldError(b *testing.B) {
	v := validator.New()
	req := &registerRequest{
		Email:                "john.doe@example.com",
		Password:             "supersecret",
		PasswordConfirmation: "different",
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(req); err == nil {
			b.Fatal("expected error")
		}
	}
}

func BenchmarkValidate_NestedArrayStructErrors(b *testing.B) {
	v := validator.New()
	req := &arrayStructRequest{
		Phones: []phone{
			{Number: ""},
			{Number: ""},
			{Number: ""},
		},
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(req); err == nil {
			b.Fatal("expected error")
		}
	}
}

func BenchmarkValidate_AttributesProvider(b *testing.B) {
	v := validator.New()
	req := &registerRequestWithAttributes{
		Email:                "john.doe@example.com",
		Password:             "",
		PasswordConfirmation: "",
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(req); err == nil {
			b.Fatal("expected error")
		}
	}
}

func BenchmarkValidate_MessagesProvider(b *testing.B) {
	v := validator.New()
	req := &createPostRequest{}

	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(req); err == nil {
			b.Fatal("expected error")
		}
	}
}
