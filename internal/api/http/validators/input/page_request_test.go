package input

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	"github.com/pauliusgin/web_app_starter-be/internal/pagination"
)

func TestValidatePageRequestInputDefaultsMissingParameters(t *testing.T) {
	pageRequest, err := ValidatePageRequestInput(PageRequestInput{})
	if err != nil {
		t.Fatalf("ValidatePageRequestInput: %v", err)
	}

	want := pagination.PageRequest{Page: primitive.DefaultPageNumber, Limit: primitive.DefaultPageLimit}
	if pageRequest != want {
		t.Errorf("page request = %+v, want %+v", pageRequest, want)
	}
}

func TestValidatePageRequestInputTrimsGivenParameters(t *testing.T) {
	pageRequest, err := ValidatePageRequestInput(PageRequestInput{Page: " 3 ", Limit: " 50"})
	if err != nil {
		t.Fatalf("ValidatePageRequestInput: %v", err)
	}

	if want := (pagination.PageRequest{Page: 3, Limit: 50}); pageRequest != want {
		t.Errorf("page request = %+v, want %+v", pageRequest, want)
	}
}

func TestValidatePageRequestInputRejectsAnInvalidPage(t *testing.T) {
	_, err := ValidatePageRequestInput(PageRequestInput{Page: "0", Limit: "10"})

	if _, ok := errors.AsType[primitive.ErrPageNumberInvalid](err); !ok {
		t.Errorf("err = %v, want ErrPageNumberInvalid", err)
	}
}

func TestValidatePageRequestInputRejectsAnInvalidLimit(t *testing.T) {
	_, err := ValidatePageRequestInput(PageRequestInput{Page: "1", Limit: "500"})

	if _, ok := errors.AsType[primitive.ErrPageLimitInvalid](err); !ok {
		t.Errorf("err = %v, want ErrPageLimitInvalid", err)
	}
}
