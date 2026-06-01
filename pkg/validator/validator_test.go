package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

type sampleDTO struct {
	Name  string `json:"name"  validate:"required,min=2,max=10"`
	Email string `json:"email" validate:"required,email"`
}

func TestValidate_Valid(t *testing.T) {
	val := validator.New()
	dto := sampleDTO{Name: "Alice", Email: "alice@example.com"}
	require.NoError(t, val.Validate(&dto))
}

func TestValidate_Required(t *testing.T) {
	val := validator.New()
	dto := sampleDTO{}
	err := val.Validate(&dto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestValidate_BadEmail(t *testing.T) {
	val := validator.New()
	dto := sampleDTO{Name: "Alice", Email: "not-an-email"}
	err := val.Validate(&dto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email")
}

func TestValidate_MinLength(t *testing.T) {
	val := validator.New()
	dto := sampleDTO{Name: "A", Email: "a@b.com"}
	err := val.Validate(&dto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestValidate_MaxLength(t *testing.T) {
	val := validator.New()
	dto := sampleDTO{Name: "TooLongName!", Email: "a@b.com"}
	err := val.Validate(&dto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}
