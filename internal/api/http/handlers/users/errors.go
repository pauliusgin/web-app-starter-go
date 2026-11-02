package users

import "fmt"

const unexpectedSignUpFailureText = "could not create the user"

const verificationTokenRequiredText = "a verification token is required"

const invalidVerificationLinkText = "this verification link is not valid or has expired"

const verificationLinkNotYoursText = "this verification link was not issued to the account you are signed in as"

const unexpectedVerificationFailureText = "could not verify the email address"

const verificationRejectedText = "rejected an email verification"

const unexpectedEmailChangeRequestFailureText = "could not request the email change"

const unexpectedPasswordChangeFailureText = "could not change the password"

const unexpectedSignInFailureText = "could not sign the user in"

const unexpectedDeletionFailureText = "could not delete the user"

const deletionRejectedText = "rejected a user deletion"

const unexpectedUserLookupFailureText = "could not load the user"

const userNotFoundText = "user not found"

// ErrVerificationTokenMissing reports a verify request that carried no token query parameter.
type ErrVerificationTokenMissing struct{}

func (e ErrVerificationTokenMissing) Error() string {
	return fmt.Sprintf("missing verification token: %s", verificationTokenRequiredText)
}
