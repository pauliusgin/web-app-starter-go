package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const presentedVerificationToken = "Kq7wL2mZ9xR4tV6yB1nH8pD3sG5jF0aC"

type inMemoryChallengeLookup struct {
	rows           []challenge.Challenge
	lookupFailure  error
	consumeFailure error
}

func newInMemoryChallengeLookup() *inMemoryChallengeLookup {
	return &inMemoryChallengeLookup{}
}

func (s *inMemoryChallengeLookup) FindByTokenHash(
	_ context.Context,
	tokenHash string,
) (challenge.Challenge, error) {
	if s.lookupFailure != nil {
		return challenge.Challenge{}, s.lookupFailure
	}

	for _, storedChallenge := range s.rows {
		if storedChallenge.TokenHash != tokenHash {
			continue
		}
		if storedChallenge.IsUsed() || storedChallenge.IsExpired(time.Now()) {
			break
		}

		return storedChallenge, nil
	}

	return challenge.Challenge{}, challenge.ErrChallengeNotFound{}
}

func (s *inMemoryChallengeLookup) MarkUsed(_ context.Context, challengeID uuid.UUID) error {
	if s.consumeFailure != nil {
		return s.consumeFailure
	}

	for index, storedChallenge := range s.rows {
		if storedChallenge.ID == challengeID {
			s.rows[index].UsedAt = fixedVerifiedAt
		}
	}

	return nil
}

func (s *inMemoryChallengeLookup) challengeByID(id uuid.UUID) (challenge.Challenge, bool) {
	for _, storedChallenge := range s.rows {
		if storedChallenge.ID == id {
			return storedChallenge, true
		}
	}

	return challenge.Challenge{}, false
}

type verifyEmailFixture struct {
	users       *inMemoryUserStore
	challenges  *inMemoryChallengeLookup
	userID      uuid.UUID
	challengeID uuid.UUID
}

func newVerifyEmailFixture(t *testing.T) *verifyEmailFixture {
	t.Helper()

	users := newInMemoryUserStore()
	users.saveUserForTest(t, testEmail)

	userID := users.savedUsers[0].ID
	challengeID := uuid.New()

	challenges := newInMemoryChallengeLookup()
	challenges.rows = append(challenges.rows, challenge.Challenge{
		ID:         challengeID,
		UserID:     userID,
		TargetType: challenge.TargetTypeEmail,
		Target:     testEmail,
		Purpose:    challenge.PurposeEmailVerification,
		TokenHash:  token.Hash(presentedVerificationToken),
		CreatedAt:  fixedCreatedAt,
		ExpiresAt:  time.Now().Add(challenge.Lifetime),
	})

	return &verifyEmailFixture{
		users:       users,
		challenges:  challenges,
		userID:      userID,
		challengeID: challengeID,
	}
}

func (f *verifyEmailFixture) dependencies() VerifyEmailDependencies {
	return VerifyEmailDependencies{
		UserWriteRepository:          f.users,
		FindAuthChallengeByTokenHash: f.challenges.FindByTokenHash,
		ConsumeAuthChallenge:         f.challenges.MarkUsed,
	}
}

func (f *verifyEmailFixture) verify(params VerifyEmailParams) error {
	return VerifyEmail(context.Background(), f.dependencies(), params)
}

func (f *verifyEmailFixture) assertNothingWasWritten(t *testing.T) {
	t.Helper()

	storedUser, err := f.users.FindUserByID(context.Background(), f.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if !storedUser.EmailVerifiedAt.IsZero() {
		t.Error("EmailVerifiedAt is stamped, want the user left unverified by a rejected request")
	}

	storedChallenge, found := f.challenges.challengeByID(f.challengeID)
	if !found {
		t.Fatal("the challenge is gone")
	}
	if storedChallenge.IsUsed() {
		t.Error("IsUsed() = true, want the challenge left unspent by a rejected request")
	}
}

func TestVerifyEmailStampsTheUserAndSpendsTheChallenge(t *testing.T) {
	fixture := newVerifyEmailFixture(t)

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})
	if err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	storedUser, err := fixture.users.FindUserByID(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.EmailVerifiedAt.IsZero() {
		t.Error("EmailVerifiedAt is the zero time, want the user stamped verified")
	}

	storedChallenge, found := fixture.challenges.challengeByID(fixture.challengeID)
	if !found {
		t.Fatal("the challenge is gone")
	}
	if !storedChallenge.IsUsed() {
		t.Error("IsUsed() = false, want the challenge spent so the link cannot be replayed")
	}
}

func TestVerifyEmailSucceedsOnASecondClick(t *testing.T) {
	fixture := newVerifyEmailFixture(t)

	params := VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	}

	if err := fixture.verify(params); err != nil {
		t.Fatalf("the first VerifyEmail: %v", err)
	}

	if err := fixture.verify(params); err != nil {
		t.Fatalf("the second VerifyEmail: %v, want a link clicked twice to succeed", err)
	}

	storedUser, err := fixture.users.FindUserByID(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.EmailVerifiedAt.IsZero() {
		t.Error("EmailVerifiedAt is the zero time, want the stamp the first click wrote left in place")
	}
}

func TestVerifyEmailRejectsALinkMatchingNothingForAnUnverifiedUser(t *testing.T) {
	fixture := newVerifyEmailFixture(t)

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: "Zx1cV4bN7mQ0wE3rT6yU9iO2pA5sD8fG",
		UserID:            fixture.userID,
	})

	if !errors.As(err, &usererrors.ErrVerificationLinkInvalid{}) {
		t.Fatalf("error = %v, want ErrVerificationLinkInvalid", err)
	}

	fixture.assertNothingWasWritten(t)
}

func TestVerifyEmailRejectsAnExpiredLinkForAnUnverifiedUser(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	fixture.challenges.rows[0].ExpiresAt = time.Now().Add(-time.Minute)

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})

	if !errors.As(err, &usererrors.ErrVerificationLinkInvalid{}) {
		t.Fatalf("error = %v, want ErrVerificationLinkInvalid", err)
	}

	fixture.assertNothingWasWritten(t)
}

func TestVerifyEmailRejectsAChallengeMintedForAnotherUser(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	fixture.challenges.rows[0].UserID = uuid.New()

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})

	if !errors.As(err, &challenge.ErrChallengeOwnedByAnotherUser{}) {
		t.Fatalf("error = %v, want ErrChallengeOwnedByAnotherUser", err)
	}

	fixture.assertNothingWasWritten(t)
}

func TestVerifyEmailTrustsTheAuthenticatedIdentityOverTheChallengesOwn(t *testing.T) {
	fixture := newVerifyEmailFixture(t)

	victimID := fixture.userID
	attackerID := uuid.New()

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            attackerID,
	})

	if !errors.As(err, &challenge.ErrChallengeOwnedByAnotherUser{}) {
		t.Fatalf("error = %v, want ErrChallengeOwnedByAnotherUser", err)
	}

	storedUser, err := fixture.users.FindUserByID(context.Background(), victimID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if !storedUser.EmailVerifiedAt.IsZero() {
		t.Error("EmailVerifiedAt is stamped, want the challenge's own user id never acted on")
	}
}

func TestVerifyEmailRejectsAChallengeMintedForAnotherPurpose(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	fixture.challenges.rows[0].Purpose = challenge.PasswordReset

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})

	purposeMismatch := challenge.ErrChallengePurposeMismatch{}
	if !errors.As(err, &purposeMismatch) {
		t.Fatalf("error = %v, want ErrChallengePurposeMismatch", err)
	}
	if purposeMismatch.Want != challenge.PurposeEmailVerification {
		t.Errorf("Want = %q, want %q", purposeMismatch.Want, challenge.PurposeEmailVerification)
	}
	if purposeMismatch.Got != challenge.PasswordReset {
		t.Errorf("Got = %q, want %q", purposeMismatch.Got, challenge.PasswordReset)
	}

	fixture.assertNothingWasWritten(t)
}

func TestVerifyEmailPropagatesALookupFailureRatherThanCallingTheLinkInvalid(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	databaseFailure := errors.New("connection reset")
	fixture.challenges.lookupFailure = challenge.ErrChallengeLookupFailed{Cause: databaseFailure}

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})

	if !errors.As(err, &challenge.ErrChallengeLookupFailed{}) {
		t.Fatalf("error = %v, want ErrChallengeLookupFailed", err)
	}
	if !errors.Is(err, databaseFailure) {
		t.Error("the underlying database failure did not survive unwrapping")
	}

	fixture.assertNothingWasWritten(t)
}

func TestVerifyEmailPropagatesAFailedVerificationStampAndLeavesTheChallengeLive(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	databaseFailure := errors.New("disk full")
	fixture.users.markVerifiedFailure = usererrors.ErrMarkEmailVerifiedFailed{Cause: databaseFailure}

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})

	if !errors.As(err, &usererrors.ErrMarkEmailVerifiedFailed{}) {
		t.Fatalf("error = %v, want ErrMarkEmailVerifiedFailed", err)
	}

	storedChallenge, found := fixture.challenges.challengeByID(fixture.challengeID)
	if !found {
		t.Fatal("the challenge is gone")
	}
	if storedChallenge.IsUsed() {
		t.Error("IsUsed() = true, want the challenge left live when the stamp failed")
	}
}

func TestVerifyEmailReportsAFailureToSpendTheChallenge(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	fixture.challenges.consumeFailure = challenge.ErrConsumeChallengeFailed{
		Cause: errors.New("deadlock detected"),
	}

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            fixture.userID,
	})

	if !errors.As(err, &challenge.ErrConsumeChallengeFailed{}) {
		t.Fatalf("error = %v, want ErrConsumeChallengeFailed", err)
	}

	storedUser, err := fixture.users.FindUserByID(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.EmailVerifiedAt.IsZero() {
		t.Error("EmailVerifiedAt is the zero time, want the stamp kept even though spending failed")
	}
}

func TestVerifyEmailPropagatesAFailedUserLookupRatherThanCallingTheLinkInvalid(t *testing.T) {
	fixture := newVerifyEmailFixture(t)
	databaseFailure := errors.New("connection refused")
	fixture.users.lookupFailure = usererrors.ErrIDLookupFailed{Cause: databaseFailure}

	err := fixture.verify(VerifyEmailParams{
		VerificationToken: "Zx1cV4bN7mQ0wE3rT6yU9iO2pA5sD8fG",
		UserID:            fixture.userID,
	})

	if !errors.As(err, &usererrors.ErrIDLookupFailed{}) {
		t.Fatalf("error = %v, want ErrIDLookupFailed", err)
	}
	if errors.As(err, &usererrors.ErrVerificationLinkInvalid{}) {
		t.Error("a failed lookup was reported as an invalid link")
	}
	if !errors.Is(err, databaseFailure) {
		t.Error("the underlying database failure did not survive unwrapping")
	}
}

type emailSwitchFixture struct {
	users       *inMemoryUserStore
	challenges  *inMemoryChallengeLookup
	eventLog    *inMemoryEventLog
	broker      *inMemoryBroker
	userID      uuid.UUID
	challengeID uuid.UUID
}

func newEmailSwitchFixture(t *testing.T) *emailSwitchFixture {
	t.Helper()

	users := newInMemoryUserStore()
	users.saveUserForTest(t, testEmail)

	userID := users.savedUsers[0].ID
	challengeID := uuid.New()

	challenges := newInMemoryChallengeLookup()
	challenges.rows = append(challenges.rows, challenge.Challenge{
		ID:         challengeID,
		UserID:     userID,
		TargetType: challenge.TargetTypeEmail,
		Target:     requestedNewEmail,
		Purpose:    challenge.PurposeEmailVerification,
		TokenHash:  token.Hash(presentedVerificationToken),
		CreatedAt:  fixedCreatedAt,
		ExpiresAt:  time.Now().Add(challenge.Lifetime),
	})

	return &emailSwitchFixture{
		users:       users,
		challenges:  challenges,
		eventLog:    newInMemoryEventLog(),
		broker:      newInMemoryBroker(),
		userID:      userID,
		challengeID: challengeID,
	}
}

func (f *emailSwitchFixture) verify(params VerifyEmailParams) error {
	recorder := func(ctx context.Context, event events.NewEvent) error {
		_, err := events.Record(ctx, events.RecordDependencies{EventWriteRepository: f.eventLog}, event)
		return err
	}

	return VerifyEmail(context.Background(), VerifyEmailDependencies{
		UserWriteRepository:          f.users,
		EventDispatcher:              events.Dispatcher{Recorder: recorder, Publisher: f.broker.publish},
		FindAuthChallengeByTokenHash: f.challenges.FindByTokenHash,
		ConsumeAuthChallenge:         f.challenges.MarkUsed,
	}, params)
}

func (f *emailSwitchFixture) validParams() VerifyEmailParams {
	return VerifyEmailParams{
		VerificationToken: presentedVerificationToken,
		UserID:            f.userID,
	}
}

func (f *emailSwitchFixture) assertEmailLeftUnchanged(t *testing.T) {
	t.Helper()

	storedUser, err := f.users.FindUserByID(context.Background(), f.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.Email != testEmail {
		t.Errorf("stored email = %q, want %q left in place by a rejected switch", storedUser.Email, testEmail)
	}
	if len(f.eventLog.recordedEvents) != 0 || len(f.broker.publishedEvents) != 0 {
		t.Errorf("recorded/published events = %d/%d, want none for a rejected switch",
			len(f.eventLog.recordedEvents), len(f.broker.publishedEvents))
	}
}

func (f *emailSwitchFixture) assertChallengeUnspent(t *testing.T) {
	t.Helper()

	storedChallenge, found := f.challenges.challengeByID(f.challengeID)
	if !found {
		t.Fatal("the challenge is gone")
	}
	if storedChallenge.IsUsed() {
		t.Error("IsUsed() = true, want the challenge left unspent so the link can be retried")
	}
}

func TestVerifyEmailSwitchesToAProvenNewAddressAndSpendsTheChallenge(t *testing.T) {
	fixture := newEmailSwitchFixture(t)

	if err := fixture.verify(fixture.validParams()); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	storedUser, err := fixture.users.FindUserByID(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.Email != requestedNewEmail {
		t.Errorf("stored email = %q, want %q", storedUser.Email, requestedNewEmail)
	}
	if !storedUser.EmailVerifiedAt.Equal(fixedVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want %v since the link proved the new address",
			storedUser.EmailVerifiedAt, fixedVerifiedAt)
	}

	storedChallenge, found := fixture.challenges.challengeByID(fixture.challengeID)
	if !found {
		t.Fatal("the challenge is gone")
	}
	if !storedChallenge.IsUsed() {
		t.Error("IsUsed() = false, want the challenge spent so the link cannot be replayed")
	}
}

func TestVerifyEmailAnnouncesASwitchWithBothAddresses(t *testing.T) {
	fixture := newEmailSwitchFixture(t)

	if err := fixture.verify(fixture.validParams()); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	if len(fixture.eventLog.recordedEvents) != 1 || len(fixture.broker.publishedEvents) != 1 {
		t.Fatalf("recorded/published events = %d/%d, want 1/1",
			len(fixture.eventLog.recordedEvents), len(fixture.broker.publishedEvents))
	}

	published := fixture.broker.publishedEvents[0]
	if published.Label != events.UserEmailChangedLabel {
		t.Errorf("Label = %q, want %q", published.Label, events.UserEmailChangedLabel)
	}

	payload := struct {
		UserID        uuid.UUID `json:"userId"`
		PreviousEmail string    `json:"previousEmail"`
		Email         string    `json:"email"`
	}{}
	if err := json.Unmarshal(published.Data, &payload); err != nil {
		t.Fatalf("unmarshalling the event data: %v", err)
	}
	if payload.UserID != fixture.userID || payload.PreviousEmail != testEmail || payload.Email != requestedNewEmail {
		t.Errorf("payload = %+v, want user %s moving from %q to %q",
			payload, fixture.userID, testEmail, requestedNewEmail)
	}
}

func TestVerifyEmailAnnouncesAVerifiedCurrentAddressAsVerifiedNotChanged(t *testing.T) {
	fixture := newEmailSwitchFixture(t)
	fixture.challenges.rows[0].Target = testEmail

	if err := fixture.verify(fixture.validParams()); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	if len(fixture.broker.publishedEvents) != 1 {
		t.Fatalf("published events = %d, want 1", len(fixture.broker.publishedEvents))
	}
	if label := fixture.broker.publishedEvents[0].Label; label != events.EmailVerifiedLabel {
		t.Errorf("Label = %q, want %q", label, events.EmailVerifiedLabel)
	}

	storedUser, err := fixture.users.FindUserByID(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("finding the user: %v", err)
	}
	if storedUser.Email != testEmail {
		t.Errorf("stored email = %q, want %q kept", storedUser.Email, testEmail)
	}
}

func TestVerifyEmailAcceptsASwitchLinkClickedTwiceOnceTheNewAddressIsVerified(t *testing.T) {
	fixture := newEmailSwitchFixture(t)

	if err := fixture.verify(fixture.validParams()); err != nil {
		t.Fatalf("the first VerifyEmail: %v", err)
	}

	if err := fixture.verify(fixture.validParams()); err != nil {
		t.Fatalf("the second VerifyEmail: %v, want a link clicked twice to succeed", err)
	}
	if len(fixture.broker.publishedEvents) != 1 {
		t.Errorf("published events = %d, want only the first click announced", len(fixture.broker.publishedEvents))
	}
}

func TestVerifyEmailRejectsASwitchToAnAddressClaimedSinceTheRequest(t *testing.T) {
	fixture := newEmailSwitchFixture(t)
	fixture.users.saveUserForTest(t, requestedNewEmail)

	err := fixture.verify(fixture.validParams())

	taken := usererrors.ErrEmailTaken{}
	if !errors.As(err, &taken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
	if taken.Email != requestedNewEmail {
		t.Errorf("ErrEmailTaken.Email = %q, want %q", taken.Email, requestedNewEmail)
	}
	fixture.assertEmailLeftUnchanged(t)
	fixture.assertChallengeUnspent(t)
}

func TestVerifyEmailPropagatesAFailedSwitchAndLeavesTheChallengeLive(t *testing.T) {
	fixture := newEmailSwitchFixture(t)
	changeFailure := usererrors.ErrChangeEmailFailed{Cause: errors.New("pq: connection refused")}
	fixture.users.changeEmailFailure = changeFailure

	err := fixture.verify(fixture.validParams())

	if err != changeFailure {
		t.Fatalf("error = %#v, want the repository's own error passed through unchanged", err)
	}
	fixture.assertEmailLeftUnchanged(t)
	fixture.assertChallengeUnspent(t)
}

func TestVerifyEmailPropagatesAFailedUserLookupBeforeSwitching(t *testing.T) {
	fixture := newEmailSwitchFixture(t)
	lookupFailure := usererrors.ErrIDLookupFailed{Cause: errors.New("pq: connection refused")}
	fixture.users.lookupFailure = lookupFailure

	err := fixture.verify(fixture.validParams())

	if err != lookupFailure {
		t.Fatalf("error = %#v, want the repository's own error passed through unchanged", err)
	}
	fixture.assertChallengeUnspent(t)
}

func TestVerifyEmailAnnouncesNoSwitchWhenSpendingTheChallengeFails(t *testing.T) {
	fixture := newEmailSwitchFixture(t)
	consumeFailure := errors.New("pq: connection refused")
	fixture.challenges.consumeFailure = consumeFailure

	err := fixture.verify(fixture.validParams())

	if !errors.Is(err, consumeFailure) {
		t.Fatalf("error = %v, want the consume failure", err)
	}
	if len(fixture.eventLog.recordedEvents) != 0 || len(fixture.broker.publishedEvents) != 0 {
		t.Errorf("recorded/published events = %d/%d, want none when the challenge is not spent",
			len(fixture.eventLog.recordedEvents), len(fixture.broker.publishedEvents))
	}
}
