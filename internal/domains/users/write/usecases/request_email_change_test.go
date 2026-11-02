package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

const otherAccountEmail = "taken@example.com"

type requestEmailChangeForTest struct {
	users      *repositories.InMemoryUserRepository
	challenges challengeLinks
	userID     uuid.UUID
}

func newRequestEmailChangeForTest(t *testing.T) requestEmailChangeForTest {
	t.Helper()

	users := repositories.NewInMemoryUserRepository()
	userID := seedUser(t, users, seedUserParams{email: testEmail, verified: true})
	seedUser(t, users, seedUserParams{email: otherAccountEmail})

	return requestEmailChangeForTest{
		users:      users,
		challenges: newChallengeLinks(),
		userID:     userID,
	}
}

func (h requestEmailChangeForTest) request(newEmail string) error {
	return RequestEmailChange(context.Background(), RequestEmailChangeDependencies{
		UserWriteRepository:   h.users,
		GenerateAuthChallenge: h.challenges.generate,
	}, RequestEmailChangeParams{UserID: h.userID, NewEmail: newEmail})
}

func (h requestEmailChangeForTest) assertEmailChangeNotRequested(t *testing.T) {
	t.Helper()

	h.challenges.assertNoneSent(t)
	if email := storedUser(t, h.users, h.userID).User.Email; email != testEmail {
		t.Errorf("stored email = %q, want %q left untouched", email, testEmail)
	}
}

func TestRequestEmailChangeSendsALinkToTheNewAddressAndKeepsTheCurrentOne(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)

	if err := harness.request(requestedNewEmail); err != nil {
		t.Fatalf("RequestEmailChange: %v", err)
	}

	wantLink := challenge.GenerateParams{
		UserID:     harness.userID,
		Target:     requestedNewEmail,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	}
	harness.challenges.assertOnlyLinkSent(t, wantLink)
	if email := storedUser(t, harness.users, harness.userID).User.Email; email != testEmail {
		t.Errorf("stored email = %q, want %q kept until the new address is verified", email, testEmail)
	}
}

func TestRequestEmailChangeRejectsAnAddressItCannotMoveTo(t *testing.T) {
	cases := []struct {
		name     string
		newEmail string
		wantErr  func(err error) bool
	}{
		{name: "current email", newEmail: testEmail, wantErr: isError[e.ErrEmailUnchanged]},
		{name: "held by another account", newEmail: otherAccountEmail, wantErr: isError[e.ErrEmailTaken]},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newRequestEmailChangeForTest(t)

			err := harness.request(testCase.newEmail)

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's rejection", err)
			}
			harness.assertEmailChangeNotRequested(t)
		})
	}
}

func TestRequestEmailChangeSendsNothingWhenRejected(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(harness *requestEmailChangeForTest)
		wantErr func(err error) bool
	}{
		{
			name: "link cannot be created",
			arrange: func(harness *requestEmailChangeForTest) {
				harness.challenges.repository.SaveFailure = challenge.ErrSaveChallengeFailed{Cause: errors.New("connection refused")}
			},
			wantErr: isError[challenge.ErrSaveChallengeFailed],
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newRequestEmailChangeForTest(t)
			testCase.arrange(&harness)

			err := harness.request(requestedNewEmail)

			if !testCase.wantErr(err) {
				t.Fatalf("error = %#v, want the case's rejection", err)
			}
			harness.assertEmailChangeNotRequested(t)
		})
	}
}

func TestRequestEmailChangeReportsAnUnknownUser(t *testing.T) {
	harness := newRequestEmailChangeForTest(t)
	harness.userID = uuid.New()

	err := harness.request(requestedNewEmail)

	if !errors.As(err, &e.ErrUserNotFound{}) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
	harness.challenges.assertNoneSent(t)
}
