package schema

import (
	"unsafe"

	"github.com/go-playground/validator/v10"
	nonstandard "github.com/go-playground/validator/v10/non-standard/validators"
)

var goValidator *validator.Validate

func init() {
	goValidator = validator.New()
	_ = goValidator.RegisterValidation("notblank", nonstandard.NotBlank)
}

type validate[T any] struct{}

func (v *validate[T]) Validate() error {
	return goValidator.Struct((*T)(unsafe.Pointer(v)))
}
