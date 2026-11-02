package input

import "fmt"

// ErrValidationRuleUnmapped reports a failed validate tag with no typed error mapped to it, a wiring bug.
type ErrValidationRuleUnmapped struct {
	Field string
	Tag   string
}

func (e ErrValidationRuleUnmapped) Error() string {
	return fmt.Sprintf("validate tag %q failed on field %q: no error is mapped to this rule", e.Tag, e.Field)
}
