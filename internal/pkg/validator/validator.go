package validator

import (
	"fmt"
	"regexp"
	"strings"

	"golang-base/internal/pkg/response"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Common validation messages
var messages = map[string]string{
	"required": "is required",
	"email":    "must be a valid email address",
	"min":      "must be at least %s characters",
	"max":      "must be at most %s characters",
	"oneof":    "must be one of: %s",
	"gte":      "must be greater than or equal to %s",
	"lte":      "must be less than or equal to %s",
	"url":      "must be a valid URL",
	"len":      "must be exactly %s characters",
}

// ValidateStruct validates a struct and returns field-level errors
func ValidateStruct(s interface{}) []response.ErrorDetail {
	var errs []response.ErrorDetail

	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return nil
	}

	for _, fe := range validationErrs {
		field := toSnakeCase(fe.Field())
		msg := buildMessage(fe)
		errs = append(errs, response.ErrorDetail{Field: field, Message: msg})
	}

	return errs
}

func buildMessage(fe validator.FieldError) string {
	tag := fe.Tag()
	param := fe.Param()

	if msg, ok := messages[tag]; ok {
		if strings.Contains(msg, "%s") && param != "" {
			return fmt.Sprintf(msg, param)
		}
		return msg
	}

	return fmt.Sprintf("failed %s validation", tag)
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Common validation regex patterns
var (
	EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	PhoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
)
