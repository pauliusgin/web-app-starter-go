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
	// ErrorCodeEmailUnchanged answers an email change request naming the address the account already has.
	ErrorCodeEmailUnchanged ErrorCode = "email_unchanged"
	// ErrorCodePasswordUnchanged answers a password change whose new password equals the current one.
	ErrorCodePasswordUnchanged ErrorCode = "password_unchanged"
	// ErrorCodeCurrentPasswordIncorrect answers a password change whose current password does not match.
	ErrorCodeCurrentPasswordIncorrect ErrorCode = "current_password_incorrect"
	// ErrorCodeInvalidCredentials answers a sign-in whose email and password match no account.
	ErrorCodeInvalidCredentials ErrorCode = "invalid_credentials"
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
	// ErrorCodeUserIDMalformed answers a path user id that is not a valid uuid.
	ErrorCodeUserIDMalformed ErrorCode = "user_id_malformed"
	// ErrorCodeUserDeletionForbidden answers an attempt to delete an account other than the caller's own.
	ErrorCodeUserDeletionForbidden ErrorCode = "user_deletion_forbidden"
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
	// ErrorCodeProfileExists answers a profile creation for a user who already has one.
	ErrorCodeProfileExists ErrorCode = "profile_exists"
	// ErrorCodeProfileNotFound answers a profile update for a user who has no profile yet.
	ErrorCodeProfileNotFound ErrorCode = "profile_not_found"
)
