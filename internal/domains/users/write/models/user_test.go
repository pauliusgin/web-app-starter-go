package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestAnonymizedEmailEmbedsTheUserID(t *testing.T) {
	id := uuid.MustParse("4c5568fa-f578-48e5-8de1-d311436e9a15")

	got := AnonymizedEmail(id)

	if want := "deleted-4c5568fa-f578-48e5-8de1-d311436e9a15"; got != want {
		t.Errorf("AnonymizedEmail = %q, want %q", got, want)
	}
}
