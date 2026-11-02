package users

import "fmt"

const unexpectedSignUpFailureText = "could not create the user"

const verificationTokenRequiredText = "a verification token is required"

const invalidVerificationLinkText = "this verification link is not valid or has expired"

const verificationLinkNotYoursText = "this verification link was not issued to the account you are signed in as"

const unexpectedVerificationFailureText = "could not verify the email address"

const unexpectedEmailChangeRequestFailureText = "could not request the email change"

const unexpectedPasswordChangeFailureText = "could not change the password"

const unexpectedPasswordAdditionFailureText = "could not add the password"

const passwordResetTokenRequiredText = "a password reset token is required"

const invalidPasswordResetLinkText = "this password reset link is not valid or has expired"

const unexpectedPasswordResetFailureText = "could not reset the password"

const unexpectedGoogleAccountAdditionFailureText = "could not add the google account"

const unexpectedSignInFailureText = "could not sign the user in"

const unexpectedGoogleSignUpFailureText = "could not create the user with google"

const unexpectedGoogleSignInFailureText = "could not sign the user in with google"

const unexpectedGoogleSignInConfigFailureText = "could not load the google sign-in config"

const unexpectedDeletionFailureText = "could not delete the user"

const userNotFoundText = "user not found"

const unexpectedUserLookupFailureText = "could not load the user"

// ErrVerificationTokenMissing reports a verify request that carried no token query parameter.
type ErrVerificationTokenMissing struct{}

func (e ErrVerificationTokenMissing) Error() string {
	return fmt.Sprintf("missing verification token: %s", verificationTokenRequiredText)
}
