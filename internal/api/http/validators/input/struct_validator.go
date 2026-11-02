package input

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// Validate tags inputs with many optional fields declare their rules with. Custom ones wrap a primitive check so the
// primitive stays the single source of its rule; the library's own email tag accepts forms our plain-address rule
// rejects. Register a new custom tag here next to the primitive it wraps.
const (
	requiredTag   = "required"
	plainEmailTag = "plain_email"
	personNameTag = "person_name"
)

// structValidator is shared because validator.Validate is safe for concurrent use and caches struct metadata.
var structValidator = newStructValidator()

func newStructValidator() *validator.Validate {
	structValidator := validator.New(validator.WithRequiredStructEnabled())

	customTags := map[string]func(string) error{
		plainEmailTag: primitive.ValidateEmail,
		personNameTag: primitive.ValidatePersonName,
	}
	for tag, check := range customTags {
		if err := structValidator.RegisterValidation(tag, tagFromPrimitive(check)); err != nil {
			panic(fmt.Sprintf("register validate tag %q: %v", tag, err))
		}
	}

	return structValidator
}

func tagFromPrimitive(check func(string) error) validator.Func {
	return func(field validator.FieldLevel) bool {
		return check(field.Field().String()) == nil
	}
}

type fieldRule struct {
	Field string
	Tag   string
}

type validateTaggedStructParams struct {
	Input      any
	RuleErrors map[fieldRule]func(value string) error
}

// validateTaggedStruct checks the validate tags of an input, reporting the first failure in field order as the
// typed error its rule maps to, so error codes stay stable whatever the library reports.
func validateTaggedStruct(params validateTaggedStructParams) error {
	err := structValidator.Struct(params.Input)
	if err == nil {
		return nil
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return err
	}

	firstFailure := fieldErrors[0]
	rule := fieldRule{Field: firstFailure.StructField(), Tag: firstFailure.Tag()}
	ruleError, mapped := params.RuleErrors[rule]
	if !mapped {
		return ErrValidationRuleUnmapped{Field: rule.Field, Tag: rule.Tag}
	}

	value, _ := firstFailure.Value().(string)

	return ruleError(value)
}
