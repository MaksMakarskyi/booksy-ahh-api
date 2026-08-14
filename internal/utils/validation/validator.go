package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

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

	_ = val.RegisterValidation("password", hasRequiredCharClasses)
	_ = val.RegisterValidation("maxbytes", withinByteLimit)

	return &CustomValidator{validator: val}
}

func hasRequiredCharClasses(fl validator.FieldLevel) bool {
	var hasLower, hasUpper, hasDigit, hasSpecial bool

	for _, r := range fl.Field().String() {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r), unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	return hasLower && hasUpper && hasDigit && hasSpecial
}

func withinByteLimit(fl validator.FieldLevel) bool {
	limit, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}

	return len(fl.Field().String()) <= limit
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
	case "password":
		return "must contain a lowercase letter, an uppercase letter, " +
			"a digit and a special character"
	case "maxbytes":
		return fmt.Sprintf("must be at most %s bytes", fe.Param())
	default:
		return "is invalid"
	}
}
