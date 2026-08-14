package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

const DateLayout = "2006-01-02"

type CustomValidator struct {
	validator *validator.Validate
}

func NewCustomValidator() *CustomValidator {
	val := validator.New(validator.WithRequiredStructEnabled())

	val.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})

	_ = val.RegisterValidation("notfuture", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}
		if _, err := time.Parse(DateLayout, value); err != nil {
			return true
		}

		return value <= time.Now().UTC().Format(DateLayout)
	})

	return &CustomValidator{validator: val}
}

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}

	return "validation failed: " + strings.Join(parts, "; ")
}

type SelfValidator interface {
	SelfValidate() []FieldError
}

func (cv *CustomValidator) Validate(i any) error {
	fields, err := cv.tagFields(i)
	if err != nil {
		return err
	}

	if sv, ok := i.(SelfValidator); ok {
		fields = append(fields, sv.SelfValidate()...)
	}

	if len(fields) == 0 {
		return nil
	}

	return &ValidationError{Fields: fields}
}

func (cv *CustomValidator) tagFields(i any) ([]FieldError, error) {
	err := cv.validator.Struct(i)
	if err == nil {
		return nil, nil
	}

	if invalid, ok := errors.AsType[*validator.InvalidValidationError](err); ok {
		return nil, invalid
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return nil, err
	}

	fields := make([]FieldError, 0, len(validationErrs))
	for _, fe := range validationErrs {
		fields = append(fields, FieldError{
			Field:   fe.Field(),
			Rule:    fe.Tag(),
			Message: message(fe),
		})
	}

	return fields, nil
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "datetime":
		return "must be a date in YYYY-MM-DD format"
	case "notfuture":
		return "cannot be in the future"
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "email":
		return "must be a valid email address"
	default:
		return "is invalid"
	}
}
