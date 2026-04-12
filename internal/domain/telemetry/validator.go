package telemetry

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError represents a structured error returned when domain rules are violated.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors is a collection of ValidationError
type ValidationErrors struct {
	Errors []ValidationError
}

func (e *ValidationErrors) Error() string {
	var msgs []string
	for _, err := range e.Errors {
		msgs = append(msgs, err.Error())
	}
	return "telemetry validation errors: " + strings.Join(msgs, " | ")
}

// Validator wraps the go-playground validator for domain-specific checks.
type Validator struct {
	validate *validator.Validate
}

// NewValidator initializes a new Telemetry Validator.
func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

// Validate checks the structural constraints of the TelemetryPayload.
func (v *Validator) Validate(p TelemetryPayload) error {
	err := v.validate.Struct(p)
	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			var ve ValidationErrors
			for _, fieldErr := range validationErrs {
				ve.Errors = append(ve.Errors, ValidationError{
					Field:   fieldErr.Field(),
					Message: fmt.Sprintf("failed on the '%s' tag", fieldErr.Tag()),
				})
			}
			return &ve
		}
		// Return unexpected parsing errors unmodified
		return fmt.Errorf("unexpected validation error: %w", err)
	}

	return nil
}
