package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

const verificationEmailRateLimitedText = "too many verification emails were requested, try again later"

const retryAfterSecondsDetail = "retryAfterSeconds"

// VerificationEmailRateLimitRules answers a verification email asked for too soon with 429 and the wait.
var VerificationEmailRateLimitRules = []RejectionRule{
	RejectWhenWithDetails[challenge.ErrIssuanceRateLimited](
		Rejection{
			Status:  http.StatusTooManyRequests,
			Code:    ErrorCodeVerificationEmailRateLimited,
			Message: verificationEmailRateLimitedText,
		},
		func(rateLimited challenge.ErrIssuanceRateLimited) map[string]any {
			return map[string]any{retryAfterSecondsDetail: retryAfterSeconds(rateLimited.RetryAfter)}
		}),
}

// WriteRetryAfterHeader sets Retry-After when err's chain holds a challenge issuance rate limit.
func WriteRetryAfterHeader(w http.ResponseWriter, err error) {
	rateLimited, isRateLimited := errors.AsType[challenge.ErrIssuanceRateLimited](err)
	if !isRateLimited {
		return
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(rateLimited.RetryAfter)))
}

// retryAfterSeconds rounds up so a client waiting the advertised seconds is never still too early.
func retryAfterSeconds(retryAfter time.Duration) int {
	return max(1, int(math.Ceil(retryAfter.Seconds())))
}
