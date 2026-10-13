package challenge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func emailVerificationParams(userID uuid.UUID) GenerateParams {
	return GenerateParams{
		UserID:     userID,
		TargetType: TargetTypeEmail,
		Target:     "someone@example.com",
		Purpose:    PurposeEmailVerification,
	}
}

func TestGenerateStoresTheHashOfTheTokenTheEventCarries(t *testing.T) {
	store := newInMemoryChallengeStore()
	captured := &capturedEvents{}
	userID := uuid.New()

	generated, err := Generate(context.Background(),
		GenerateDependencies{
			WriteRepository: store,
			EventDispatcher: events.Dispatcher{Recorder: captured.recorder(), Publisher: captured.publisher()},
		},
		emailVerificationParams(userID))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	storedChallenges := store.challengesForUser(userID)
	if len(storedChallenges) != 1 {
		t.Fatalf("stored rows = %d, want 1", len(storedChallenges))
	}
	storedChallenge := storedChallenges[0]

	plaintext := tokenFromEventData(t, captured.onlyPublished(t))
	if plaintext == "" {
		t.Fatal("the published event carries no token, want the plaintext for the link")
	}

	if storedChallenge.TokenHash != token.Hash(plaintext) {
		t.Errorf("stored hash = %q, want the hash of the published plaintext %q",
			storedChallenge.TokenHash, token.Hash(plaintext))
	}
	if storedChallenge.TokenHash == plaintext {
		t.Error("the repository received the plaintext, want only its hash")
	}
	if strings.Contains(fmt.Sprintf("%+v", generated), plaintext) {
		t.Error("the returned value carries the plaintext, want the event to be its only path")
	}
}

func TestGenerateExpiresOneLifetimeAfterCreation(t *testing.T) {
	store := newInMemoryChallengeStore()
	captured := &capturedEvents{}
	userID := uuid.New()

	issuedNoEarlierThan := time.Now()
	generated, err := Generate(context.Background(),
		GenerateDependencies{
			WriteRepository: store,
			EventDispatcher: events.Dispatcher{Recorder: captured.recorder(), Publisher: captured.publisher()},
		},
		emailVerificationParams(userID))
	issuedNoLaterThan := time.Now()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	earliestExpiry := issuedNoEarlierThan.Add(Lifetime)
	latestExpiry := issuedNoLaterThan.Add(Lifetime)
	if generated.ExpiresAt.Before(earliestExpiry) || generated.ExpiresAt.After(latestExpiry) {
		t.Errorf("ExpiresAt = %s, want within [%s, %s]", generated.ExpiresAt, earliestExpiry, latestExpiry)
	}
}

func TestGenerateKeepsTheTokenOutOfTheRecordedPayload(t *testing.T) {
	store := newInMemoryChallengeStore()
	captured := &capturedEvents{}
	userID := uuid.New()

	_, err := Generate(context.Background(),
		GenerateDependencies{
			WriteRepository: store,
			EventDispatcher: events.Dispatcher{Recorder: captured.recorder(), Publisher: captured.publisher()},
		},
		emailVerificationParams(userID))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	recorded := captured.onlyRecorded(t)
	plaintext := tokenFromEventData(t, recorded)
	if plaintext == "" {
		t.Fatal("the recorded event's Data carries no token, want it for the broker payload")
	}

	if recorded.RedactedData == nil {
		t.Fatal("the recorded event carries no RedactedData, want the redacted payload the event log stores")
	}

	redactedPayload := string(recorded.RedactedData)
	if strings.Contains(redactedPayload, plaintext) {
		t.Errorf("RedactedData carries the plaintext token: %s", redactedPayload)
	}
	if !strings.Contains(string(recorded.Data), plaintext) {
		t.Error("Data does not carry the plaintext token, want it for the link")
	}
	if !strings.Contains(redactedPayload, "someone@example.com") {
		t.Errorf("RedactedData dropped the target as well as the token: %s", redactedPayload)
	}
}

func TestGenerateRejectsAnIncompleteRequestBeforeMintingAnything(t *testing.T) {
	userID := uuid.New()

	cases := []struct {
		name        string
		mutate      func(params *GenerateParams)
		isWantedErr func(err error) bool
		wantErrName string
	}{
		{
			name:        "no target",
			mutate:      func(params *GenerateParams) { params.Target = "" },
			isWantedErr: func(err error) bool { return errors.As(err, &ErrTargetMissing{}) },
			wantErrName: "ErrTargetMissing",
		},
		{
			name:   "no target type",
			mutate: func(params *GenerateParams) { params.TargetType = "" },
			isWantedErr: func(err error) bool {
				return errors.As(err, &ErrTargetTypeMissing{})
			},
			wantErrName: "ErrTargetTypeMissing",
		},
		{
			name:        "no purpose",
			mutate:      func(params *GenerateParams) { params.Purpose = "" },
			isWantedErr: func(err error) bool { return errors.As(err, &ErrPurposeMissing{}) },
			wantErrName: "ErrPurposeMissing",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newInMemoryChallengeStore()
			captured := &capturedEvents{}
			params := emailVerificationParams(userID)
			testCase.mutate(&params)

			generated, err := Generate(context.Background(),
				GenerateDependencies{
					WriteRepository: store,
					EventDispatcher: events.Dispatcher{Recorder: captured.recorder(), Publisher: captured.publisher()},
				},
				params)

			if !testCase.isWantedErr(err) {
				t.Fatalf("error = %v, want %s", err, testCase.wantErrName)
			}
			if generated != (Generated{}) {
				t.Errorf("generated = %+v, want the zero value on failure", generated)
			}
			if len(store.rows) != 0 {
				t.Errorf("stored rows = %d, want nothing stored for a bad request",
					len(store.rows))
			}
			if len(captured.recorded) != 0 || len(captured.published) != 0 {
				t.Errorf("recorded %d and published %d events, want none for a bad request",
					len(captured.recorded), len(captured.published))
			}
		})
	}
}

func TestGeneratePassesAStorageFailureThroughAndAnnouncesNothing(t *testing.T) {
	cause := errors.New("pq: connection refused")
	saveFailure := ErrSaveChallengeFailed{Cause: cause}
	store := newInMemoryChallengeStore()
	store.saveFailure = saveFailure
	captured := &capturedEvents{}

	generated, err := Generate(context.Background(),
		GenerateDependencies{
			WriteRepository: store,
			EventDispatcher: events.Dispatcher{Recorder: captured.recorder(), Publisher: captured.publisher()},
		},
		emailVerificationParams(uuid.New()))

	if err != saveFailure {
		t.Fatalf("error = %#v, want the repository's own error passed through unchanged", err)
	}

	saveFailed := ErrSaveChallengeFailed{}
	if !errors.As(err, &saveFailed) {
		t.Fatalf("error = %v, want ErrSaveChallengeFailed", err)
	}
	if generated != (Generated{}) {
		t.Errorf("generated = %+v, want the zero value on failure", generated)
	}
	if len(store.rows) != 0 {
		t.Errorf("stored rows = %d, want nothing stored when the insert fails", len(store.rows))
	}
	if len(captured.recorded) != 0 || len(captured.published) != 0 {
		t.Errorf("recorded %d and published %d events, want none when the insert fails",
			len(captured.recorded), len(captured.published))
	}
}
