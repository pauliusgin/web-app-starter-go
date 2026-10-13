package users

import "fmt"

const unexpectedSignupFailureText = "could not create the user"

const invalidVerificationLinkText = "this verification link is not valid or has expired"

const verificationLinkNotYoursText = "this verification link was not issued to the account you are signed in as"

const unexpectedVerificationFailureText = "could not verify the email address"

const verificationRejectedText = "rejected an email verification"

const unexpectedEmailChangeRequestFailureText = "could not request the email change"

const unexpectedPasswordChangeFailureText = "could not change the password"

const unexpectedSignInFailureText = "could not sign the user in"

const unexpectedDeletionFailureText = "could not delete the user"

const deletionRejectedText = "rejected a user deletion"

const userIDMalformedText = "the user id in the path is not a valid uuid"

const deletionNotYoursText = "you can only delete your own account"

const deletedUserNotFoundText = "user not found"

// ErrUserIDMalformed reports a path user id that does not parse as a uuid.
type ErrUserIDMalformed struct {
	RawUserID string
	Cause     error
}

func (e ErrUserIDMalformed) Error() string {
	return fmt.Sprintf("user id %q: %s: %s", e.RawUserID, userIDMalformedText, e.Cause)
}

func (e ErrUserIDMalformed) Unwrap() error {
	return e.Cause
}
