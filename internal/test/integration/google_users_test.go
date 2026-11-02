package integration

import (
	"database/sql"
	"errors"
	"slices"
	"testing"

	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

func TestGoogleUserSignsUpVerifiedAndSignsInOnlyWithGoogle(t *testing.T) {
	world := newIntegrationWorld(t)

	signedUp, err := world.signUpGoogle("accountant@example.com")
	if err != nil {
		t.Fatalf("SignUpUserGoogle: %v", err)
	}

	readModel := world.getUserByID(t, signedUp.User.ID).User
	if readModel.EmailVerifiedAt == nil || readModel.SigninAt == nil {
		t.Errorf("EmailVerifiedAt/SigninAt = %v/%v, want both stamped at signup", readModel.EmailVerifiedAt, readModel.SigninAt)
	}

	var passwordHash sql.NullString
	var googleSubject string
	err = world.database.QueryRowContext(world.ctx,
		`SELECT password_hash, google_subject FROM users WHERE id = $1`, signedUp.User.ID).Scan(&passwordHash, &googleSubject)
	if err != nil {
		t.Fatalf("reading the stored row: %v", err)
	}
	if passwordHash.Valid || googleSubject != integrationGoogleSubject {
		t.Errorf("password_hash/google_subject = %v/%q, want NULL/%q", passwordHash, googleSubject, integrationGoogleSubject)
	}

	signedIn, err := world.signInGoogle("accountant@example.com")
	if err != nil || signedIn.User.ID != signedUp.User.ID {
		t.Fatalf("SignInUserGoogle = %s, %v, want user %s", signedIn.User.ID, err, signedUp.User.ID)
	}

	_, passwordSignInErr := world.signIn(userusecases.SignInUserParams{Email: "accountant@example.com", Password: testPassword})
	if !errors.As(passwordSignInErr, &usererrors.ErrCredentialsInvalid{}) {
		t.Errorf("password sign-in err = %v, want ErrCredentialsInvalid for a google-only account", passwordSignInErr)
	}
}

func TestGoogleSignInRefusesAPasswordAccountHoldingTheEmail(t *testing.T) {
	world := newIntegrationWorld(t)
	world.signUp(t, "accountant@example.com")

	_, signUpErr := world.signUpGoogle("accountant@example.com")
	if !errors.As(signUpErr, &usererrors.ErrEmailTaken{}) {
		t.Errorf("google signup err = %v, want ErrEmailTaken", signUpErr)
	}

	_, signInErr := world.signInGoogle("accountant@example.com")
	if !errors.As(signInErr, &usererrors.ErrGoogleAccountNotLinked{}) {
		t.Errorf("google sign-in err = %v, want ErrGoogleAccountNotLinked", signInErr)
	}
}

func TestDeletedGoogleUserFreesTheGoogleAccountForANewSignup(t *testing.T) {
	world := newIntegrationWorld(t)
	signedUp, err := world.signUpGoogle("accountant@example.com")
	if err != nil {
		t.Fatalf("SignUpUserGoogle: %v", err)
	}

	if err := world.deleteUser(userusecases.DeleteUserParams{UserID: signedUp.User.ID}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	var googleSubject sql.NullString
	err = world.database.QueryRowContext(world.ctx,
		`SELECT google_subject FROM users WHERE id = $1`, signedUp.User.ID).Scan(&googleSubject)
	if err != nil {
		t.Fatalf("reading the deleted row: %v", err)
	}
	if googleSubject.Valid {
		t.Errorf("google_subject = %q on the deleted row, want NULL", googleSubject.String)
	}

	if _, err := world.signInGoogle("accountant@example.com"); !errors.As(err, &usererrors.ErrGoogleAccountNotFound{}) {
		t.Errorf("signing in as the deleted google user err = %v, want ErrGoogleAccountNotFound", err)
	}
	if _, err := world.signUpGoogle("accountant@example.com"); err != nil {
		t.Errorf("signing up the google account again: %v", err)
	}
}

func TestGoogleUserAddsAPasswordAndSignsInWithEither(t *testing.T) {
	world := newIntegrationWorld(t)
	signedUp, err := world.signUpGoogle("accountant@example.com")
	if err != nil {
		t.Fatalf("SignUpUserGoogle: %v", err)
	}

	err = world.addPassword(userusecases.AddPasswordParams{UserID: signedUp.User.ID, Password: testPassword})
	if err != nil {
		t.Fatalf("AddPassword: %v", err)
	}

	passwordSignIn, err := world.signIn(userusecases.SignInUserParams{Email: "accountant@example.com", Password: testPassword})
	if err != nil || passwordSignIn.User.ID != signedUp.User.ID {
		t.Fatalf("SignInUser = %s, %v, want user %s", passwordSignIn.User.ID, err, signedUp.User.ID)
	}
	googleSignIn, err := world.signInGoogle("accountant@example.com")
	if err != nil || googleSignIn.User.ID != signedUp.User.ID {
		t.Fatalf("SignInUserGoogle = %s, %v, want user %s", googleSignIn.User.ID, err, signedUp.User.ID)
	}

	again := world.addPassword(userusecases.AddPasswordParams{UserID: signedUp.User.ID, Password: "another-ledger-password"})
	if !errors.As(again, &usererrors.ErrPasswordAlreadySet{}) {
		t.Errorf("second AddPassword err = %v, want ErrPasswordAlreadySet", again)
	}
}

func TestUserReadModelListsTheSignInMethodsTheStoredCredentialsAllow(t *testing.T) {
	world := newIntegrationWorld(t)
	basicUser := world.signUp(t, "basic@example.com")
	googleUser, err := world.signUpGoogle("google@example.com")
	if err != nil {
		t.Fatalf("SignUpUserGoogle: %v", err)
	}

	assertSignInMethods(t, world.getUserByID(t, basicUser.User.ID).User.SignInMethods, readmodels.SignInMethodEmailPassword)
	assertSignInMethods(t, world.getUserByID(t, googleUser.User.ID).User.SignInMethods, readmodels.SignInMethodGoogle)

	err = world.addPassword(userusecases.AddPasswordParams{UserID: googleUser.User.ID, Password: testPassword})
	if err != nil {
		t.Fatalf("AddPassword: %v", err)
	}

	assertSignInMethods(t, world.getUserByID(t, googleUser.User.ID).User.SignInMethods,
		readmodels.SignInMethodEmailPassword, readmodels.SignInMethodGoogle)
}

func assertSignInMethods(t *testing.T, got []readmodels.SignInMethod, want ...readmodels.SignInMethod) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("SignInMethods = %v, want %v", got, want)
	}
}
