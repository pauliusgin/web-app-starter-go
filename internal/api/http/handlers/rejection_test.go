package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const unexpectedFailureTextForTest = "could not do the thing"

const rejectedLogTextForTest = "rejected the thing"

const fixedMessageForTest = "you may not do the thing"

type errFirstRuleForTest struct{}

func (e errFirstRuleForTest) Error() string {
	return "first rule matched"
}

type errSecondRuleForTest struct {
	Detail string
}

func (e errSecondRuleForTest) Error() string {
	return "second rule matched: " + e.Detail
}

var firstRulesForTest = []RejectionRule{
	RejectWhen[errFirstRuleForTest](Rejection{Status: http.StatusConflict, Code: ErrorCodeEmailTaken}),
}

var secondRulesForTest = []RejectionRule{
	RejectWhen[errFirstRuleForTest](Rejection{Status: http.StatusTeapot, Code: ErrorCodeInternal}),
	RejectWhen[errSecondRuleForTest](Rejection{
		Status:  http.StatusForbidden,
		Code:    ErrorCodeUserDeletionForbidden,
		Message: fixedMessageForTest,
	}),
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logs := &bytes.Buffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return logs
}

func writeRejectionForTest(t *testing.T, params WriteRejectionParams) (int, ErrorResponse) {
	t.Helper()

	recorder := httptest.NewRecorder()
	WriteRejection(recorder, params)

	body := ErrorResponse{}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return recorder.Code, body
}

func TestSelectRejectionPicksTheFirstMatchingRuleAcrossRuleSets(t *testing.T) {
	rejection, matched := SelectRejection(errFirstRuleForTest{}, firstRulesForTest, secondRulesForTest)

	if !matched {
		t.Fatal("matched = false, want the first rule set to match")
	}
	if rejection.Status != http.StatusConflict || rejection.Code != ErrorCodeEmailTaken {
		t.Errorf("rejection = %+v, want the first rule set's 409 email_taken", rejection)
	}
}

func TestSelectRejectionMatchesAWrappedError(t *testing.T) {
	wrapped := fmt.Errorf("signing up: %w", errSecondRuleForTest{Detail: "inner"})

	rejection, matched := SelectRejection(wrapped, secondRulesForTest)

	if !matched {
		t.Fatal("matched = false, want the wrapped error to match")
	}
	if rejection.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rejection.Status, http.StatusForbidden)
	}
}

func TestSelectRejectionUsesTheMatchedErrorTextWhenNoMessageIsFixed(t *testing.T) {
	matchedError := errFirstRuleForTest{}
	wrapped := fmt.Errorf("outer context: %w", matchedError)

	rejection, _ := SelectRejection(wrapped, firstRulesForTest)

	if rejection.Message != matchedError.Error() {
		t.Errorf("Message = %q, want the matched error's own text %q", rejection.Message, matchedError.Error())
	}
}

func TestSelectRejectionKeepsAFixedMessage(t *testing.T) {
	rejection, _ := SelectRejection(errSecondRuleForTest{Detail: "secret"}, secondRulesForTest)

	if rejection.Message != fixedMessageForTest {
		t.Errorf("Message = %q, want %q", rejection.Message, fixedMessageForTest)
	}
}

func TestSelectRejectionReportsNoMatch(t *testing.T) {
	_, matched := SelectRejection(errors.New("disk on fire"), firstRulesForTest, secondRulesForTest)

	if matched {
		t.Error("matched = true, want false for an error no rule names")
	}
}

func TestWriteRejectionAnswersAnUnmatchedErrorWithAGenericInternalError(t *testing.T) {
	logs := captureLogs(t)

	status, body := writeRejectionForTest(t, WriteRejectionParams{
		Err:                   errors.New("disk on fire"),
		RuleSets:              [][]RejectionRule{firstRulesForTest},
		UnexpectedFailureText: unexpectedFailureTextForTest,
	})

	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if body.Code != ErrorCodeInternal || body.Message != unexpectedFailureTextForTest {
		t.Errorf("body = %+v, want internal_error with the generic text", body)
	}
	if !strings.Contains(logs.String(), "level=ERROR") || !strings.Contains(logs.String(), "disk on fire") {
		t.Errorf("logs = %q, want the cause logged at ERROR", logs.String())
	}
}

func TestWriteRejectionAnswersAMatchedError(t *testing.T) {
	captureLogs(t)

	status, body := writeRejectionForTest(t, WriteRejectionParams{
		Err:                   errFirstRuleForTest{},
		RuleSets:              [][]RejectionRule{firstRulesForTest},
		UnexpectedFailureText: unexpectedFailureTextForTest,
	})

	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	if body.Code != ErrorCodeEmailTaken || body.Message != (errFirstRuleForTest{}).Error() {
		t.Errorf("body = %+v, want email_taken with the matched error's text", body)
	}
}

func TestWriteRejectionLogsMatchedRejectionsOnlyWhenAskedTo(t *testing.T) {
	cases := []struct {
		name            string
		rejectedLogText string
		wantWarning     bool
	}{
		{name: "rejected log text set", rejectedLogText: rejectedLogTextForTest, wantWarning: true},
		{name: "rejected log text empty"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureLogs(t)

			writeRejectionForTest(t, WriteRejectionParams{
				Err:                   errFirstRuleForTest{},
				RuleSets:              [][]RejectionRule{firstRulesForTest},
				UnexpectedFailureText: unexpectedFailureTextForTest,
				RejectedLogText:       testCase.rejectedLogText,
			})

			warned := strings.Contains(logs.String(), "level=WARN")
			if warned != testCase.wantWarning {
				t.Errorf("warning logged = %t, want %t (logs %q)", warned, testCase.wantWarning, logs.String())
			}
		})
	}
}

func TestWriteRejectionAnswersAbsentDetailsAsNull(t *testing.T) {
	captureLogs(t)

	recorder := httptest.NewRecorder()
	WriteRejection(recorder, WriteRejectionParams{
		Err:                   errFirstRuleForTest{},
		RuleSets:              [][]RejectionRule{firstRulesForTest},
		UnexpectedFailureText: unexpectedFailureTextForTest,
	})

	body := map[string]any{}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}
	details, present := body["details"]
	if !present || details != nil {
		t.Errorf("details = %v (present %t), want present as null", details, present)
	}
}
