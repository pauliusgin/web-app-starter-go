// Package userrepositorytest holds the behaviour every user write repository implementation must share.
package userrepositorytest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

// RunUserRepositoryContractParams builds a fresh, empty repository for every contract case.
type RunUserRepositoryContractParams struct {
	NewRepository func(t *testing.T) repositories.UserRepository
}

// RunUserRepositoryContract runs the shared cases, so the in-memory fake cannot drift from the SQL.
func RunUserRepositoryContract(t *testing.T, params RunUserRepositoryContractParams) {
	cases := []struct {
		name string
		run  func(t *testing.T, harness contractHarness)
	}{
		{name: "saved user is found by id and email, signed in but unverified", run: savedUserIsFoundSignedInButUnverified},
		{name: "saving a held email fails", run: savingAHeldEmailFails},
		{name: "stamps and password change are stored", run: stampsAndPasswordChangeAreStored},
		{name: "changing to a held email is refused", run: changingToAHeldEmailIsRefused},
		{name: "soft-deleted user is gone and frees the email", run: softDeletedUserIsGoneAndFreesTheEmail},
		{name: "saved google user is found by subject, verified and without a password", run: savedGoogleUserIsFoundVerifiedWithoutPassword},
		{name: "saving a held google subject or email fails", run: savingAHeldGoogleSubjectOrEmailFails},
		{name: "soft-deleted google user frees the google subject", run: softDeletedGoogleUserFreesTheGoogleSubject},
		{name: "google user gains a password once, keeping the google subject", run: googleUserGainsAPasswordOnce},
		{name: "adding a password to a password or unknown user is refused", run: addingAPasswordToANonGoogleUserIsRefused},
		{name: "password user gains a google account once, verified, keeping the password", run: passwordUserGainsAGoogleAccountOnce},
		{name: "adding a google account keeps an earlier email verification", run: addingAGoogleAccountKeepsEarlierVerification},
		{name: "adding a held or unknown user's google account is refused", run: addingAHeldOrUnknownGoogleAccountIsRefused},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t, contractHarness{repository: params.NewRepository(t), ctx: context.Background()})
		})
	}
}

type contractHarness struct {
	repository repositories.UserRepository
	ctx        context.Context
}

func (h contractHarness) save(t *testing.T, email string) models.User {
	t.Helper()

	saved, err := h.repository.SaveUser(h.ctx, models.NewUser{ID: uuid.New(), Email: email, PasswordHash: "$argon2id$original"})
	if err != nil {
		t.Fatalf("Save(%q): %v", email, err)
	}

	return saved
}

func assertUserNotFound(t *testing.T, operation string, err error) {
	t.Helper()

	if !errors.As(err, &e.ErrUserNotFound{}) {
		t.Errorf("%s err = %v, want ErrUserNotFound", operation, err)
	}
}

func savedUserIsFoundSignedInButUnverified(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")

	byID, err := harness.repository.FindUserByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	byEmail, err := harness.repository.FindUserByEmail(harness.ctx, "accountant@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if byID.ID != saved.ID || byEmail.ID != saved.ID {
		t.Errorf("found ids = %s / %s, want %s", byID.ID, byEmail.ID, saved.ID)
	}
	if byID.IsEmailVerified() {
		t.Error("IsEmailVerified() = true, want a fresh user unverified")
	}
	if byID.SigninAt.IsZero() {
		t.Error("SigninAt is zero, want signup to count as the first sign-in")
	}

	credentials, err := harness.repository.FindUserCredentialsByEmail(harness.ctx, "accountant@example.com")
	if err != nil {
		t.Fatalf("FindUserCredentialsByEmail: %v", err)
	}
	if credentials.PasswordHash != "$argon2id$original" {
		t.Errorf("PasswordHash = %q, want the saved hash", credentials.PasswordHash)
	}

	_, unknownErr := harness.repository.FindUserByID(harness.ctx, uuid.New())
	assertUserNotFound(t, "finding an unknown id", unknownErr)
}

func savingAHeldEmailFails(t *testing.T, harness contractHarness) {
	harness.save(t, "accountant@example.com")

	_, err := harness.repository.SaveUser(harness.ctx, models.NewUser{
		ID:           uuid.New(),
		Email:        "accountant@example.com",
		PasswordHash: "$argon2id$other",
	})

	if !errors.As(err, &e.ErrSaveUserFailed{}) {
		t.Errorf("err = %v, want ErrSaveUserFailed", err)
	}
}

func stampsAndPasswordChangeAreStored(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")

	verified, err := harness.repository.MarkEmailVerified(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}
	if !verified.IsEmailVerified() {
		t.Error("IsEmailVerified() = false after MarkEmailVerified")
	}
	signedIn, err := harness.repository.MarkSignedIn(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("MarkSignedIn: %v", err)
	}
	if !signedIn.SigninAt.After(saved.SigninAt) {
		t.Errorf("SigninAt = %v, want it moved past the signup stamp %v", signedIn.SigninAt, saved.SigninAt)
	}

	err = harness.repository.ChangePassword(harness.ctx, models.ChangePasswordParams{ID: saved.ID, PasswordHash: "$argon2id$new"})
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	credentials, err := harness.repository.FindUserCredentialsByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserCredentialsByID: %v", err)
	}
	if credentials.PasswordHash != "$argon2id$new" {
		t.Errorf("PasswordHash = %q, want the changed hash", credentials.PasswordHash)
	}

	_, unknownErr := harness.repository.MarkEmailVerified(harness.ctx, uuid.New())
	assertUserNotFound(t, "verifying an unknown user", unknownErr)
}

func changingToAHeldEmailIsRefused(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")
	harness.save(t, "bookkeeper@example.com")

	_, err := harness.repository.ChangeEmail(harness.ctx, models.ChangeEmailParams{ID: saved.ID, Email: "bookkeeper@example.com"})
	if !errors.As(err, &e.ErrEmailTaken{}) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}

	changed, err := harness.repository.ChangeEmail(harness.ctx, models.ChangeEmailParams{ID: saved.ID, Email: "auditor@example.com"})
	if err != nil {
		t.Fatalf("ChangeEmail to a free address: %v", err)
	}
	if changed.Email != "auditor@example.com" || !changed.IsEmailVerified() {
		t.Errorf("changed = %+v, want the new email stamped verified", changed)
	}
}

func softDeletedUserIsGoneAndFreesTheEmail(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")
	deleteParams := models.SoftDeleteUserParams{
		ID:                  saved.ID,
		AnonymizedEmail:     "deleted-" + saved.ID.String(),
		AnonymizedFirstName: "Deleted",
		AnonymizedLastName:  "User",
	}

	if err := harness.repository.SoftDeleteUser(harness.ctx, deleteParams); err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}

	_, byIDErr := harness.repository.FindUserByID(harness.ctx, saved.ID)
	assertUserNotFound(t, "finding by id", byIDErr)
	_, credentialsErr := harness.repository.FindUserCredentialsByEmail(harness.ctx, "accountant@example.com")
	assertUserNotFound(t, "finding credentials by email", credentialsErr)
	assertUserNotFound(t, "deleting again", harness.repository.SoftDeleteUser(harness.ctx, deleteParams))

	harness.save(t, "accountant@example.com")
}

const contractGoogleSubject = "109876543210987654321"

func (h contractHarness) saveGoogle(t *testing.T, params models.NewGoogleUserParams) models.User {
	t.Helper()

	saved, err := h.repository.SaveGoogleUser(h.ctx, models.NewGoogle(params))
	if err != nil {
		t.Fatalf("SaveGoogleUser(%q): %v", params.Email, err)
	}

	return saved
}

func savedGoogleUserIsFoundVerifiedWithoutPassword(t *testing.T, harness contractHarness) {
	saved := harness.saveGoogle(t, models.NewGoogleUserParams{
		Email:         "accountant@example.com",
		GoogleSubject: contractGoogleSubject,
	})
	if !saved.IsEmailVerified() || saved.SigninAt.IsZero() {
		t.Errorf("saved = %+v, want a google user stored verified and signed in", saved)
	}

	bySubject, err := harness.repository.FindUserByGoogleSubject(harness.ctx, contractGoogleSubject)
	if err != nil {
		t.Fatalf("FindUserByGoogleSubject: %v", err)
	}
	if bySubject.ID != saved.ID || !bySubject.IsEmailVerified() {
		t.Errorf("found = %+v, want the saved verified user %s", bySubject, saved.ID)
	}

	credentials, err := harness.repository.FindUserCredentialsByEmail(harness.ctx, "accountant@example.com")
	if err != nil {
		t.Fatalf("FindUserCredentialsByEmail: %v", err)
	}
	if credentials.HasPassword() {
		t.Errorf("PasswordHash = %q, want none for a google user", credentials.PasswordHash)
	}

	_, unknownErr := harness.repository.FindUserByGoogleSubject(harness.ctx, "unknown-subject")
	assertUserNotFound(t, "finding an unknown google subject", unknownErr)

	harness.save(t, "bookkeeper@example.com")
	_, passwordUserErr := harness.repository.FindUserByGoogleSubject(harness.ctx, "")
	assertUserNotFound(t, "finding a password user by an empty google subject", passwordUserErr)
}

func savingAHeldGoogleSubjectOrEmailFails(t *testing.T, harness contractHarness) {
	harness.saveGoogle(t, models.NewGoogleUserParams{Email: "accountant@example.com", GoogleSubject: contractGoogleSubject})
	harness.save(t, "bookkeeper@example.com")

	cases := []struct {
		name   string
		params models.NewGoogleUserParams
	}{
		{name: "held google subject", params: models.NewGoogleUserParams{Email: "auditor@example.com", GoogleSubject: contractGoogleSubject}},
		{name: "email held by a password user", params: models.NewGoogleUserParams{Email: "bookkeeper@example.com", GoogleSubject: "other-subject"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := harness.repository.SaveGoogleUser(harness.ctx, models.NewGoogle(testCase.params))

			if !errors.As(err, &e.ErrSaveUserFailed{}) {
				t.Errorf("err = %v, want ErrSaveUserFailed", err)
			}
		})
	}
}

func softDeletedGoogleUserFreesTheGoogleSubject(t *testing.T, harness contractHarness) {
	saved := harness.saveGoogle(t, models.NewGoogleUserParams{Email: "accountant@example.com", GoogleSubject: contractGoogleSubject})

	err := harness.repository.SoftDeleteUser(harness.ctx, models.SoftDeleteUserParams{
		ID:                  saved.ID,
		AnonymizedEmail:     "deleted-" + saved.ID.String(),
		AnonymizedFirstName: "Deleted",
		AnonymizedLastName:  "User",
	})
	if err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}

	_, bySubjectErr := harness.repository.FindUserByGoogleSubject(harness.ctx, contractGoogleSubject)
	assertUserNotFound(t, "finding a deleted google user", bySubjectErr)

	harness.saveGoogle(t, models.NewGoogleUserParams{Email: "accountant@example.com", GoogleSubject: contractGoogleSubject})
}

func assertPasswordAlreadySet(t *testing.T, operation string, err error) {
	t.Helper()

	if !errors.As(err, &e.ErrPasswordAlreadySet{}) {
		t.Errorf("%s err = %v, want ErrPasswordAlreadySet", operation, err)
	}
}

func googleUserGainsAPasswordOnce(t *testing.T, harness contractHarness) {
	saved := harness.saveGoogle(t, models.NewGoogleUserParams{Email: "accountant@example.com", GoogleSubject: contractGoogleSubject})

	err := harness.repository.AddPassword(harness.ctx, models.AddPasswordParams{ID: saved.ID, PasswordHash: "$argon2id$added"})
	if err != nil {
		t.Fatalf("AddPassword: %v", err)
	}

	credentials, err := harness.repository.FindUserCredentialsByEmail(harness.ctx, "accountant@example.com")
	if err != nil {
		t.Fatalf("FindUserCredentialsByEmail: %v", err)
	}
	if credentials.PasswordHash != "$argon2id$added" || credentials.GoogleSubject != contractGoogleSubject {
		t.Errorf("credentials = %+v, want the added hash next to the kept google subject", credentials)
	}

	again := harness.repository.AddPassword(harness.ctx, models.AddPasswordParams{ID: saved.ID, PasswordHash: "$argon2id$second"})
	assertPasswordAlreadySet(t, "adding a second password", again)
	afterRetry, err := harness.repository.FindUserCredentialsByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserCredentialsByID: %v", err)
	}
	if afterRetry.PasswordHash != "$argon2id$added" {
		t.Errorf("PasswordHash = %q, want the first added hash kept", afterRetry.PasswordHash)
	}
}

func addingAPasswordToANonGoogleUserIsRefused(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")

	passwordUserErr := harness.repository.AddPassword(harness.ctx, models.AddPasswordParams{ID: saved.ID, PasswordHash: "$argon2id$added"})
	assertPasswordAlreadySet(t, "adding a password to a password user", passwordUserErr)
	credentials, err := harness.repository.FindUserCredentialsByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserCredentialsByID: %v", err)
	}
	if credentials.PasswordHash != "$argon2id$original" || credentials.HasGoogleAccount() {
		t.Errorf("credentials = %+v, want the original hash and no google subject", credentials)
	}

	unknownErr := harness.repository.AddPassword(harness.ctx, models.AddPasswordParams{ID: uuid.New(), PasswordHash: "$argon2id$added"})
	assertPasswordAlreadySet(t, "adding a password to an unknown user", unknownErr)
}

func assertGoogleAccountAlreadyConnected(t *testing.T, operation string, err error) {
	t.Helper()

	if !errors.As(err, &e.ErrGoogleAccountAlreadyConnected{}) {
		t.Errorf("%s err = %v, want ErrGoogleAccountAlreadyConnected", operation, err)
	}
}

func passwordUserGainsAGoogleAccountOnce(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")

	err := harness.repository.AddGoogleAccount(harness.ctx, models.AddGoogleAccountParams{ID: saved.ID, GoogleSubject: contractGoogleSubject})
	if err != nil {
		t.Fatalf("AddGoogleAccount: %v", err)
	}

	googleUser, err := harness.repository.FindUserByGoogleSubject(harness.ctx, contractGoogleSubject)
	if err != nil {
		t.Fatalf("FindUserByGoogleSubject: %v", err)
	}
	if googleUser.ID != saved.ID || !googleUser.IsEmailVerified() {
		t.Errorf("google user = %+v, want user %s with the email stamped verified", googleUser, saved.ID)
	}
	credentials, err := harness.repository.FindUserCredentialsByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserCredentialsByID: %v", err)
	}
	if credentials.PasswordHash != "$argon2id$original" {
		t.Errorf("PasswordHash = %q, want the original hash kept", credentials.PasswordHash)
	}

	again := harness.repository.AddGoogleAccount(harness.ctx, models.AddGoogleAccountParams{ID: saved.ID, GoogleSubject: "999999999999999999999"})
	assertGoogleAccountAlreadyConnected(t, "adding a second google account", again)
	afterRetry, err := harness.repository.FindUserCredentialsByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserCredentialsByID: %v", err)
	}
	if afterRetry.GoogleSubject != contractGoogleSubject {
		t.Errorf("GoogleSubject = %q, want the first google subject kept", afterRetry.GoogleSubject)
	}
}

func addingAGoogleAccountKeepsEarlierVerification(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "accountant@example.com")
	verified, err := harness.repository.MarkEmailVerified(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}

	err = harness.repository.AddGoogleAccount(harness.ctx, models.AddGoogleAccountParams{ID: saved.ID, GoogleSubject: contractGoogleSubject})
	if err != nil {
		t.Fatalf("AddGoogleAccount: %v", err)
	}

	googleUser, err := harness.repository.FindUserByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if !googleUser.EmailVerifiedAt.Equal(verified.EmailVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want the earlier stamp %v kept", googleUser.EmailVerifiedAt, verified.EmailVerifiedAt)
	}
}

func addingAHeldOrUnknownGoogleAccountIsRefused(t *testing.T, harness contractHarness) {
	harness.saveGoogle(t, models.NewGoogleUserParams{Email: "owner@example.com", GoogleSubject: contractGoogleSubject})
	saved := harness.save(t, "accountant@example.com")

	heldErr := harness.repository.AddGoogleAccount(harness.ctx, models.AddGoogleAccountParams{ID: saved.ID, GoogleSubject: contractGoogleSubject})
	if !errors.As(heldErr, &e.ErrGoogleAccountTaken{}) {
		t.Errorf("adding a held google subject err = %v, want ErrGoogleAccountTaken", heldErr)
	}
	credentials, err := harness.repository.FindUserCredentialsByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserCredentialsByID: %v", err)
	}
	if credentials.HasGoogleAccount() {
		t.Errorf("credentials = %+v, want no google subject", credentials)
	}

	unknownErr := harness.repository.AddGoogleAccount(harness.ctx, models.AddGoogleAccountParams{ID: uuid.New(), GoogleSubject: "999999999999999999999"})
	assertGoogleAccountAlreadyConnected(t, "adding a google account to an unknown user", unknownErr)
}
