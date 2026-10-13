package brevo

import (
	"errors"
	"testing"
)

func TestBrevoSendErrorsUnwrapToTheFailureBrevoOrTheNetworkReported(t *testing.T) {
	cases := []struct {
		name  string
		build func(cause error) error
	}{
		{
			name:  "rejected",
			build: func(cause error) error { return ErrSendRejected{Cause: cause} },
		},
		{
			name:  "unanswered",
			build: func(cause error) error { return ErrSendUnanswered{Cause: cause} },
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cause := errors.New("the underlying failure")

			err := testCase.build(cause)

			if !errors.Is(err, cause) {
				t.Errorf("error = %v, want it to unwrap to the cause", err)
			}
			if errors.Unwrap(err) != cause {
				t.Errorf("one unwrap reached %v, want the cause directly", errors.Unwrap(err))
			}
		})
	}
}
