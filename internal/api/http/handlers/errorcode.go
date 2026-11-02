package handlers

// ErrorCode is the stable, machine-readable reason a request failed; clients branch on it, never on Message.
type ErrorCode string

const (
	// ErrorCodeInternal answers every unexpected server-side failure.
	ErrorCodeInternal ErrorCode = "internal_error"
	// ErrorCodeRequestBodyInvalid answers a request body that is empty, malformed or the wrong shape.
	ErrorCodeRequestBodyInvalid ErrorCode = "request_body_invalid"
	// ErrorCodeRequestBodyTooLarge answers a request body over the decoder's byte limit.
	ErrorCodeRequestBodyTooLarge ErrorCode = "request_body_too_large"
	// ErrorCodeUnsupportedContentType answers a request whose Content-Type is not application/json.
	ErrorCodeUnsupportedContentType ErrorCode = "unsupported_content_type"
	// ErrorCodeEmailInvalid answers an email that is not a plain address.
	ErrorCodeEmailInvalid ErrorCode = "email_invalid"
	// ErrorCodePasswordMissing answers a request that carried no password.
	ErrorCodePasswordMissing ErrorCode = "password_missing"
	// ErrorCodePasswordTooShort answers a password under the minimum length.
	ErrorCodePasswordTooShort ErrorCode = "password_too_short"
	// ErrorCodePasswordTooLong answers a password over the maximum length.
	ErrorCodePasswordTooLong ErrorCode = "password_too_long"
	// ErrorCodePasswordContainsWhitespace answers a password holding whitespace.
	ErrorCodePasswordContainsWhitespace ErrorCode = "password_contains_whitespace"
	// ErrorCodeEmailTaken answers a signup for an email that already has an account.
	ErrorCodeEmailTaken ErrorCode = "email_taken"
	// ErrorCodeEmailAlreadyVerified answers a verification email resend for an account whose email is verified.
	ErrorCodeEmailAlreadyVerified ErrorCode = "email_already_verified"
	// ErrorCodeVerificationEmailRateLimited answers a verification email asked for before the limits allow another.
	ErrorCodeVerificationEmailRateLimited ErrorCode = "verification_email_rate_limited"
	// ErrorCodeEmailUnchanged answers an email change request naming the address the account already has.
	ErrorCodeEmailUnchanged ErrorCode = "email_unchanged"
	// ErrorCodePasswordUnchanged answers a password change whose new password equals the current one.
	ErrorCodePasswordUnchanged ErrorCode = "password_unchanged"
	// ErrorCodeCurrentPasswordIncorrect answers a password change whose current password does not match.
	ErrorCodeCurrentPasswordIncorrect ErrorCode = "current_password_incorrect"
	// ErrorCodeInvalidCredentials answers a sign-in whose email and password match no account.
	ErrorCodeInvalidCredentials ErrorCode = "invalid_credentials"
	// ErrorCodeGoogleIDTokenMissing answers a google signup or sign-in that carried no ID token.
	ErrorCodeGoogleIDTokenMissing ErrorCode = "google_id_token_missing"
	// ErrorCodeGoogleIDTokenInvalid answers a google ID token that is malformed, forged, expired or for another app.
	ErrorCodeGoogleIDTokenInvalid ErrorCode = "google_id_token_invalid"
	// ErrorCodeGoogleEmailUnverified answers a google account whose email google has not verified.
	ErrorCodeGoogleEmailUnverified ErrorCode = "google_email_unverified"
	// ErrorCodeGoogleAccountTaken answers a google signup for a google account that is already signed up.
	ErrorCodeGoogleAccountTaken ErrorCode = "google_account_taken"
	// ErrorCodeGoogleAccountNotFound answers a google sign-in for a google account and email with no account.
	ErrorCodeGoogleAccountNotFound ErrorCode = "google_account_not_found"
	// ErrorCodeGoogleAccountNotLinked answers a google sign-in whose email belongs to a password account.
	ErrorCodeGoogleAccountNotLinked ErrorCode = "google_account_not_linked"
	// ErrorCodePasswordNotSet answers a password change for an account that signs in with google only.
	ErrorCodePasswordNotSet ErrorCode = "password_not_set"
	// ErrorCodePasswordAlreadySet answers a password addition for an account that already has a password.
	ErrorCodePasswordAlreadySet ErrorCode = "password_already_set"
	// ErrorCodeGoogleAccountNotConnected answers a password addition for an account not signed up with google.
	ErrorCodeGoogleAccountNotConnected ErrorCode = "google_account_not_connected"
	// ErrorCodeGoogleAccountAlreadyConnected answers a google account addition for an account that already has one.
	ErrorCodeGoogleAccountAlreadyConnected ErrorCode = "google_account_already_connected"
	// ErrorCodeGoogleEmailMismatch answers a google account addition whose google email differs from the account's.
	ErrorCodeGoogleEmailMismatch ErrorCode = "google_email_mismatch"
	// ErrorCodeAccessTokenMissing answers a protected request with no Authorization header.
	ErrorCodeAccessTokenMissing ErrorCode = "access_token_missing"
	// ErrorCodeAccessTokenExpired answers a protected request whose access token has expired.
	ErrorCodeAccessTokenExpired ErrorCode = "access_token_expired"
	// ErrorCodeAccessTokenInvalid answers every other access token rejection, collapsed so account state does not leak.
	ErrorCodeAccessTokenInvalid ErrorCode = "access_token_invalid"
	// ErrorCodeRefreshTokenInvalid answers every refresh token rejection, collapsed so account state does not leak.
	ErrorCodeRefreshTokenInvalid ErrorCode = "refresh_token_invalid"
	// ErrorCodeVerificationTokenMissing answers an email verification request with no token.
	ErrorCodeVerificationTokenMissing ErrorCode = "verification_token_missing"
	// ErrorCodeVerificationLinkInvalid answers a verification link that is used, expired or for another purpose.
	ErrorCodeVerificationLinkInvalid ErrorCode = "verification_link_invalid"
	// ErrorCodeVerificationLinkNotOwned answers a verification link issued to a different account.
	ErrorCodeVerificationLinkNotOwned ErrorCode = "verification_link_not_owned"
	// ErrorCodePasswordResetTokenMissing answers a password reset request with no token.
	ErrorCodePasswordResetTokenMissing ErrorCode = "password_reset_token_missing"
	// ErrorCodePasswordResetLinkInvalid answers every reset link rejection, collapsed so account state does not leak.
	ErrorCodePasswordResetLinkInvalid ErrorCode = "password_reset_link_invalid"
	// ErrorCodeUserNotFound answers a request naming a user that does not exist.
	ErrorCodeUserNotFound ErrorCode = "user_not_found"
	// ErrorCodeUploadPurposeInvalid answers an upload whose purpose is missing or unknown.
	ErrorCodeUploadPurposeInvalid ErrorCode = "upload_purpose_invalid"
	// ErrorCodeUploadFilesMissing answers an upload carrying no file.
	ErrorCodeUploadFilesMissing ErrorCode = "upload_files_missing"
	// ErrorCodeUploadTooManyFiles answers an upload carrying more files than its purpose accepts in one request.
	ErrorCodeUploadTooManyFiles ErrorCode = "upload_too_many_files"
	// ErrorCodeUploadFileTypeUnsupported answers a file whose format the server or the purpose does not accept.
	ErrorCodeUploadFileTypeUnsupported ErrorCode = "upload_file_type_unsupported"
	// ErrorCodeUploadFileTooLarge answers a file over the size limit of its format.
	ErrorCodeUploadFileTooLarge ErrorCode = "upload_file_too_large"
	// ErrorCodeUploadImageDimensionsTooLarge answers an image wider or taller than the pixel limit.
	ErrorCodeUploadImageDimensionsTooLarge ErrorCode = "upload_image_dimensions_too_large"
	// ErrorCodeUploadFileUnreadable answers an image whose content cannot be decoded.
	ErrorCodeUploadFileUnreadable ErrorCode = "upload_file_unreadable"
	// ErrorCodeFirstNameMissing answers a profile request with an empty or blank first name.
	ErrorCodeFirstNameMissing ErrorCode = "first_name_missing"
	// ErrorCodeLastNameMissing answers a profile request with an empty or blank last name.
	ErrorCodeLastNameMissing ErrorCode = "last_name_missing"
	// ErrorCodeFirstNameInvalid answers a profile request whose first name breaks the person-name rules.
	ErrorCodeFirstNameInvalid ErrorCode = "first_name_invalid"
	// ErrorCodeLastNameInvalid answers a profile request whose last name breaks the person-name rules.
	ErrorCodeLastNameInvalid ErrorCode = "last_name_invalid"
	// ErrorCodeProfilePictureUploadInvalid answers a picture upload id that is malformed, unknown, not the caller's,
	// or not a profile picture.
	ErrorCodeProfilePictureUploadInvalid ErrorCode = "profile_picture_upload_invalid"
	// ErrorCodeProfileExists answers a profile creation for a user who already has one.
	ErrorCodeProfileExists ErrorCode = "profile_exists"
	// ErrorCodeProfileNotFound answers a profile update for a user who has no profile yet.
	ErrorCodeProfileNotFound ErrorCode = "profile_not_found"
	// ErrorCodePageInvalid answers a list request whose page query parameter is not a whole number in range.
	ErrorCodePageInvalid ErrorCode = "page_invalid"
	// ErrorCodeLimitInvalid answers a list request whose limit query parameter is not a whole number in range.
	ErrorCodeLimitInvalid ErrorCode = "limit_invalid"
)
