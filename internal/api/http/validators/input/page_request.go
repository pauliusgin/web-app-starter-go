package input

import (
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	"github.com/pauliusgin/web_app_starter-be/internal/pagination"
)

// PageRequestInput carries the raw page and limit query parameters of a list request; either may be empty.
type PageRequestInput struct {
	Page  string
	Limit string
}

// ValidatePageRequestInput parses the trimmed page and limit, defaulting the empty ones.
func ValidatePageRequestInput(rawInput PageRequestInput) (pagination.PageRequest, error) {
	page, err := primitive.ParsePageNumber(strings.TrimSpace(rawInput.Page))
	if err != nil {
		return pagination.PageRequest{}, err
	}

	limit, err := primitive.ParsePageLimit(strings.TrimSpace(rawInput.Limit))
	if err != nil {
		return pagination.PageRequest{}, err
	}

	return pagination.PageRequest{Page: page, Limit: limit}, nil
}
