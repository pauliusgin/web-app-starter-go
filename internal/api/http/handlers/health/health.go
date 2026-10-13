// Package health answers the liveness probe.
package health

import (
	"net/http"
)

// Report returns a handler answering 200 with an empty body.
func Report() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
