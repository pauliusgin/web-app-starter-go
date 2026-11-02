package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

func TestVerificationEmailRateLimitRulesAnswerTheWaitInWholeSeconds(t *testing.T) {
	cases := []struct {
		name       string
		retryAfter time.Duration
		wantWait   int
	}{
		{name: "whole seconds", retryAfter: 20 * time.Second, wantWait: 20},
		{name: "fraction rounds up", retryAfter: 20*time.Second + time.Millisecond, wantWait: 21},
		{name: "under a second waits one", retryAfter: time.Millisecond, wantWait: 1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := fmt.Errorf("wrapped: %w", challenge.ErrIssuanceRateLimited{RetryAfter: testCase.retryAfter})

			rejection, matched := SelectRejection(err, VerificationEmailRateLimitRules)

			if !matched {
				t.Fatal("matched = false, want the rate limit rule to match")
			}
			if rejection.Status != http.StatusTooManyRequests || rejection.Code != ErrorCodeVerificationEmailRateLimited {
				t.Errorf("rejection = %+v, want 429 verification_email_rate_limited", rejection)
			}
			if rejection.Details[retryAfterSecondsDetail] != testCase.wantWait {
				t.Errorf("retryAfterSeconds = %v, want %d", rejection.Details[retryAfterSecondsDetail], testCase.wantWait)
			}
		})
	}
}

func TestWriteRetryAfterHeaderSetsTheHeaderOnlyForARateLimit(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantHeader string
	}{
		{name: "rate limited", err: challenge.ErrIssuanceRateLimited{RetryAfter: 1500 * time.Millisecond}, wantHeader: "2"},
		{name: "other failure", err: errors.New("boom")},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			WriteRetryAfterHeader(recorder, testCase.err)

			if got := recorder.Header().Get("Retry-After"); got != testCase.wantHeader {
				t.Errorf("Retry-After = %q, want %q", got, testCase.wantHeader)
			}
		})
	}
}
