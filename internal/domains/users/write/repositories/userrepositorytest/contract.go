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
	saved := harness.save(t, "member@example.com")

	byID, err := harness.repository.FindUserByID(harness.ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	byEmail, err := harness.repository.FindUserByEmail(harness.ctx, "member@example.com")
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

	credentials, err := harness.repository.FindUserCredentialsByEmail(harness.ctx, "member@example.com")
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
	harness.save(t, "member@example.com")

	_, err := harness.repository.SaveUser(harness.ctx, models.NewUser{
		ID:           uuid.New(),
		Email:        "member@example.com",
		PasswordHash: "$argon2id$other",
	})

	if !errors.As(err, &e.ErrSaveUserFailed{}) {
		t.Errorf("err = %v, want ErrSaveUserFailed", err)
	}
}

func stampsAndPasswordChangeAreStored(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "member@example.com")

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
	saved := harness.save(t, "member@example.com")
	harness.save(t, "other-member@example.com")

	_, err := harness.repository.ChangeEmail(harness.ctx, models.ChangeEmailParams{ID: saved.ID, Email: "other-member@example.com"})
	if !errors.As(err, &e.ErrEmailTaken{}) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}

	changed, err := harness.repository.ChangeEmail(harness.ctx, models.ChangeEmailParams{ID: saved.ID, Email: "new-address@example.com"})
	if err != nil {
		t.Fatalf("ChangeEmail to a free address: %v", err)
	}
	if changed.Email != "new-address@example.com" || !changed.IsEmailVerified() {
		t.Errorf("changed = %+v, want the new email stamped verified", changed)
	}
}

func softDeletedUserIsGoneAndFreesTheEmail(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "member@example.com")
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
	_, credentialsErr := harness.repository.FindUserCredentialsByEmail(harness.ctx, "member@example.com")
	assertUserNotFound(t, "finding credentials by email", credentialsErr)
	assertUserNotFound(t, "deleting again", harness.repository.SoftDeleteUser(harness.ctx, deleteParams))

	harness.save(t, "member@example.com")
}
