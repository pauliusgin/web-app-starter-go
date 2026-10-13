package events

// Label is the stable wire name of an event kind, stored in the log and used as a routing key.
type Label string

// UserSignedUpLabel labels the event dispatched when a new account is created.
const UserSignedUpLabel Label = "user.signed_up"

// UserSignedInLabel labels the event dispatched when an account signs in with its credentials.
const UserSignedInLabel Label = "user.signed_in"

// UserDeletedLabel labels the event dispatched when an account is soft-deleted.
const UserDeletedLabel Label = "user.deleted"

// EmailVerifiedLabel labels the event dispatched when an account's email is verified.
const EmailVerifiedLabel Label = "user.email_verified"

// UserEmailChangedLabel labels the event dispatched when an account switches to a newly verified email.
const UserEmailChangedLabel Label = "user.email_changed"

// UserPasswordChangedLabel labels the event dispatched when an account replaces its password.
const UserPasswordChangedLabel Label = "user.password_changed"

// AuthChallengeGeneratedLabel labels the event dispatched when a verification challenge is minted.
const AuthChallengeGeneratedLabel Label = "auth_challenge.generated"

// MessageSentLabel labels the event dispatched when a delivery provider accepts a message to a user.
const MessageSentLabel Label = "message.sent"

// MessageFailedLabel labels the event dispatched when a message to a user could not be handed off.
const MessageFailedLabel Label = "message.failed"

// UploadCreatedLabel labels the event dispatched when a user's file is stored and recorded as pending.
const UploadCreatedLabel Label = "upload.created"

// UploadDeletedLabel labels the event dispatched when an upload and its stored files are removed.
const UploadDeletedLabel Label = "upload.deleted"

// ProfileCreatedLabel labels the event dispatched when a user records their profile.
const ProfileCreatedLabel Label = "profile.created"

// ProfileUpdatedLabel labels the event dispatched when a user replaces their profile's names.
const ProfileUpdatedLabel Label = "profile.updated"
