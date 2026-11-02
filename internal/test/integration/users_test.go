package integration

import (
	"errors"
	"testing"

	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	userusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

func TestSignedUpUserVerifiesTheEmailAndSignsIn(t *testing.T) {
	world := newIntegrationWorld(t)
	signedUp := world.signUp(t, "member@example.com")

	beforeVerification := world.getUserByID(t, signedUp.User.ID).User
	if beforeVerification.EmailVerifiedAt != nil {
		t.Fatalf("EmailVerifiedAt = %v for a fresh user, want NULL", beforeVerification.EmailVerifiedAt)
	}

	err := world.verifyEmail(userusecases.VerifyEmailParams{
		VerificationToken: world.latestChallengeToken(t, "member@example.com"),
		UserID:            signedUp.User.ID,
	})
	if err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	if _, err := world.signIn(userusecases.SignInUserParams{Email: "member@example.com", Password: testPassword}); err != nil {
		t.Fatalf("SignInUser: %v", err)
	}

	afterSignIn := world.getUserByID(t, signedUp.User.ID).User
	if afterSignIn.EmailVerifiedAt == nil {
		t.Error("EmailVerifiedAt = NULL, want the verification stamped")
	}
	if afterSignIn.SigninAt == nil || !afterSignIn.SigninAt.After(*beforeVerification.SigninAt) {
		t.Errorf("SigninAt = %v, want it moved past the signup stamp %v", afterSignIn.SigninAt, beforeVerification.SigninAt)
	}
}

func TestSignUpRejectsAnEmailAnotherAccountHolds(t *testing.T) {
	world := newIntegrationWorld(t)
	world.signUp(t, "member@example.com")

	_, err := world.signUpUser(userusecases.SignUpUserParams{Email: "member@example.com", Password: testPassword})

	if !errors.As(err, &usererrors.ErrEmailTaken{}) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}

func TestEmailChangeSwitchesTheAddressOnceTheNewOneIsProven(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	currentEmail := world.getUserByID(t, userID).User.Email

	if err := world.requestEmailChange(userusecases.RequestEmailChangeParams{
		UserID:   userID,
		NewEmail: "new-address@example.com",
	}); err != nil {
		t.Fatalf("RequestEmailChange: %v", err)
	}
	if email := world.getUserByID(t, userID).User.Email; email != currentEmail {
		t.Fatalf("Email = %q before the link is clicked, want %q kept", email, currentEmail)
	}

	err := world.verifyEmail(userusecases.VerifyEmailParams{
		VerificationToken: world.latestChallengeToken(t, "new-address@example.com"),
		UserID:            userID,
	})
	if err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	if email := world.getUserByID(t, userID).User.Email; email != "new-address@example.com" {
		t.Errorf("Email = %q, want the proven new address", email)
	}
	if _, err := world.signIn(userusecases.SignInUserParams{Email: "new-address@example.com", Password: testPassword}); err != nil {
		t.Errorf("SignInUser with the new address: %v", err)
	}
}

func TestEmailChangeRejectsAnAddressAnotherAccountHolds(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)
	world.signUp(t, "new-address@example.com")

	err := world.requestEmailChange(userusecases.RequestEmailChangeParams{UserID: userID, NewEmail: "new-address@example.com"})

	if !errors.As(err, &usererrors.ErrEmailTaken{}) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}

func TestUserReadModelCarriesTheProfileNames(t *testing.T) {
	world := newIntegrationWorld(t)
	userID := world.verifiedUserID(t)

	if _, err := world.createProfile(profileusecases.CreateProfileParams{
		UserID:    userID,
		FirstName: "Ada",
		LastName:  "Lovelace",
	}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	user := world.getUserByID(t, userID).User
	assertTextColumn(t, "FirstName", user.FirstName, "Ada")
	assertTextColumn(t, "LastName", user.LastName, "Lovelace")
}

func TestDeletedUserIsGoneAnonymizedAndFreesTheEmail(t *testing.T) {
	world := newIntegrationWorld(t)
	signedUp := world.signUp(t, "member@example.com")
	if _, err := world.createProfile(profileusecases.CreateProfileParams{
		UserID:    signedUp.User.ID,
		FirstName: "Ada",
		LastName:  "Lovelace",
	}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if err := world.deleteUser(userusecases.DeleteUserParams{UserID: signedUp.User.ID}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if _, err := world.userReadRepository.GetUserReadModelByID(world.ctx, signedUp.User.ID); !errors.As(err, &usererrors.ErrUserNotFound{}) {
		t.Errorf("reading the deleted user err = %v, want ErrUserNotFound", err)
	}
	_, signInErr := world.signIn(userusecases.SignInUserParams{Email: "member@example.com", Password: testPassword})
	if !errors.As(signInErr, &usererrors.ErrCredentialsInvalid{}) {
		t.Errorf("signing in as the deleted user err = %v, want ErrInvalidCredentials", signInErr)
	}

	var storedEmail, firstName, lastName string
	err := world.database.QueryRowContext(world.ctx, `
		SELECT u.email, p.first_name, p.last_name
		FROM users u JOIN profiles p ON p.id = u.id
		WHERE u.id = $1`, signedUp.User.ID).Scan(&storedEmail, &firstName, &lastName)
	if err != nil {
		t.Fatalf("reading the anonymized rows: %v", err)
	}
	if storedEmail == "member@example.com" || firstName == "Ada" || lastName == "Lovelace" {
		t.Errorf("stored row = %q %q %q, want the personal data anonymized", storedEmail, firstName, lastName)
	}

	world.signUp(t, "member@example.com")
}
