package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

type contactInput struct {
	Email    string `validate:"omitempty,plain_email"`
	Nickname string `validate:"omitempty,person_name"`
	Name     string `validate:"required"`
}

var contactRuleErrors = map[fieldRule]func(value string) error{
	{Field: "Email", Tag: plainEmailTag}: func(value string) error { return primitive.ErrEmailInvalid{Email: value} },
	{Field: "Nickname", Tag: personNameTag}: func(value string) error {
		return primitive.ErrFirstNameInvalid{Cause: primitive.ErrPersonNameMalformed{Name: value}}
	},
}

func validateContact(contact contactInput) error {
	return validateTaggedStruct(validateTaggedStructParams{Input: contact, RuleErrors: contactRuleErrors})
}

func TestValidateTaggedStructAcceptsAnInputPassingEveryTag(t *testing.T) {
	err := validateContact(contactInput{Email: "member@example.com", Nickname: "Ada", Name: "Ada Lovelace"})

	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

func TestValidateTaggedStructSkipsOmittedOptionalFields(t *testing.T) {
	if err := validateContact(contactInput{Name: "Ada Lovelace"}); err != nil {
		t.Errorf("err = %v, want nil for blank optional fields", err)
	}
}

func TestValidateTaggedStructReportsTheMappedErrorOfTheFirstFailingField(t *testing.T) {
	err := validateContact(contactInput{Email: "Ada <ada@example.com>", Nickname: "Ada3", Name: "Ada Lovelace"})

	var emailInvalid primitive.ErrEmailInvalid
	if !errors.As(err, &emailInvalid) || emailInvalid.Email != "Ada <ada@example.com>" {
		t.Errorf("err = %v, want ErrEmailInvalid carrying the rejected email", err)
	}
}

func TestValidateTaggedStructReportsACustomTagFailureAsItsPrimitiveError(t *testing.T) {
	err := validateContact(contactInput{Nickname: "-Ada", Name: "Ada Lovelace"})

	if !errors.As(err, &primitive.ErrFirstNameInvalid{}) {
		t.Errorf("err = %v, want ErrFirstNameInvalid", err)
	}
}

func TestValidateTaggedStructReportsAnUnmappedRuleAsAWiringBug(t *testing.T) {
	err := validateContact(contactInput{})

	var unmapped ErrValidationRuleUnmapped
	if !errors.As(err, &unmapped) || unmapped.Field != "Name" || unmapped.Tag != requiredTag {
		t.Errorf("err = %v, want ErrValidationRuleUnmapped for Name/%s", err, requiredTag)
	}
}
