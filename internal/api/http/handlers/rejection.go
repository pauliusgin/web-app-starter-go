package handlers

import (
	"errors"
	"log/slog"
	"net/http"
)

// Rejection is the status, code, optional fixed message and details a failed request answers with.
type Rejection struct {
	Status  int
	Code    ErrorCode
	Message string
	Details map[string]any
}

// RejectionRule pairs an error type with the rejection a failure of that type answers with.
type RejectionRule struct {
	findMatchingError func(err error) (error, bool)
	describeDetails   func(matchedError error) map[string]any
	rejection         Rejection
}

// RejectWhen builds a rule that answers with rejection when err's chain holds an E.
func RejectWhen[E error](rejection Rejection) RejectionRule {
	return RejectionRule{
		findMatchingError: func(err error) (error, bool) {
			return errors.AsType[E](err)
		},
		rejection: rejection,
	}
}

// RejectWhenWithDetails builds a RejectWhen rule whose details are read off the matched E.
func RejectWhenWithDetails[E error](rejection Rejection, describeDetails func(matched E) map[string]any) RejectionRule {
	rule := RejectWhen[E](rejection)
	rule.describeDetails = func(matchedError error) map[string]any {
		return describeDetails(matchedError.(E))
	}

	return rule
}

// SelectRejection returns the rejection of the first rule matching err, its message and details filled in.
func SelectRejection(err error, ruleSets ...[]RejectionRule) (Rejection, bool) {
	for _, rules := range ruleSets {
		for _, rule := range rules {
			matchedError, matches := rule.findMatchingError(err)
			if !matches {
				continue
			}

			return rule.rejectionFor(matchedError), true
		}
	}

	return Rejection{}, false
}

func (r RejectionRule) rejectionFor(matchedError error) Rejection {
	rejection := r.rejection
	if rejection.Message == "" {
		rejection.Message = matchedError.Error()
	}
	if r.describeDetails != nil {
		rejection.Details = r.describeDetails(matchedError)
	}

	return rejection
}

// WriteRejectionParams is the failure to answer, the rules to answer it with and the 500 text to log.
type WriteRejectionParams struct {
	Err                   error
	RuleSets              [][]RejectionRule
	UnexpectedFailureText string
}

// WriteRejection answers Err with the first matching rule's rejection unlogged, or a logged internal_error.
func WriteRejection(w http.ResponseWriter, params WriteRejectionParams) {
	rejection, matched := SelectRejection(params.Err, params.RuleSets...)
	if !matched {
		WriteUnexpectedFailure(w, WriteUnexpectedFailureParams{
			Err:         params.Err,
			FailureText: params.UnexpectedFailureText,
		})
		return
	}

	WriteJSON(w, rejection.Status, ErrorResponse{
		Code:    rejection.Code,
		Message: rejection.Message,
		Details: rejection.Details,
	})
}

// WriteUnexpectedFailureParams is the failure to log and the generic text the client gets instead.
type WriteUnexpectedFailureParams struct {
	Err         error
	FailureText string
}

// WriteUnexpectedFailure logs Err and answers 500 internal_error with only the generic FailureText.
func WriteUnexpectedFailure(w http.ResponseWriter, params WriteUnexpectedFailureParams) {
	slog.Error(params.FailureText, "error", params.Err)
	WriteJSON(w, http.StatusInternalServerError, ErrorResponse{
		Code:    ErrorCodeInternal,
		Message: params.FailureText,
	})
}
