package primitive

import "strconv"

// DefaultPageNumber is the page a list request gets when it names none.
const DefaultPageNumber = 1

// MaxPageNumber is the highest page a list request may ask for; with MaxPageLimit it keeps offsets within int32.
const MaxPageNumber = 1_000

// DefaultPageLimit is how many items a list request gets per page when it names no limit.
const DefaultPageLimit = 20

// MaxPageLimit is the most items a list request may ask for per page.
const MaxPageLimit = 100

// ParsePageNumber reads a page number, defaulting an empty one to DefaultPageNumber.
func ParsePageNumber(rawPage string) (int, error) {
	page, ok := parseBoundedPositiveInt(parseBoundedPositiveIntParams{
		raw:          rawPage,
		defaultValue: DefaultPageNumber,
		maxValue:     MaxPageNumber,
	})
	if !ok {
		return 0, ErrPageNumberInvalid{RawPage: rawPage}
	}

	return page, nil
}

// ParsePageLimit reads a per-page limit, defaulting an empty one to DefaultPageLimit.
func ParsePageLimit(rawLimit string) (int, error) {
	limit, ok := parseBoundedPositiveInt(parseBoundedPositiveIntParams{
		raw:          rawLimit,
		defaultValue: DefaultPageLimit,
		maxValue:     MaxPageLimit,
	})
	if !ok {
		return 0, ErrPageLimitInvalid{RawLimit: rawLimit}
	}

	return limit, nil
}

type parseBoundedPositiveIntParams struct {
	raw          string
	defaultValue int
	maxValue     int
}

func parseBoundedPositiveInt(params parseBoundedPositiveIntParams) (int, bool) {
	if params.raw == "" {
		return params.defaultValue, true
	}

	value, err := strconv.Atoi(params.raw)
	if err != nil || value < 1 || value > params.maxValue {
		return 0, false
	}

	return value, true
}
