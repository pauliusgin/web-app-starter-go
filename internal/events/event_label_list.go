package events

// Label is the stable wire name of an event kind, stored in the log and used as a routing key.
type Label string

// UserSignedUpLabel labels the event dispatched when a new account is created.
const UserSignedUpLabel Label = "user.signed_up"

// UserSignedInLabel labels the event dispatched when an account signs in with its credentials.
const UserSignedInLabel Label = "user.signed_in"

// UserDeletedLabel labels the event dispatched when an account is soft-deleted.
const UserDeletedLabel Label = "user.deleted"

// UserEmailVerifiedLabel labels the event dispatched when an account's email is verified.
const UserEmailVerifiedLabel Label = "user.email_verified"

// UserEmailChangedLabel labels the event dispatched when an account switches to a newly verified email.
const UserEmailChangedLabel Label = "user.email_changed"

// UserPasswordChangedLabel labels the event dispatched when an account replaces its password.
const UserPasswordChangedLabel Label = "user.password_changed"

// UserPasswordAddedLabel labels the event dispatched when a google-only account adds a password.
const UserPasswordAddedLabel Label = "user.password_added"

// UserPasswordResetLabel labels the event dispatched when an account sets a new password through a reset link.
const UserPasswordResetLabel Label = "user.password_reset"

// UserGoogleAccountAddedLabel labels the event dispatched when a password account connects a google account.
const UserGoogleAccountAddedLabel Label = "user.google_account_added"

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

// CompanyCreatedLabel labels the event dispatched when a user records a company they invoice from.
const CompanyCreatedLabel Label = "company.created"

// CompanyUpdatedLabel labels the event dispatched when a user changes details of a company.
const CompanyUpdatedLabel Label = "company.updated"

// CompanyDeletedLabel labels the event dispatched when a user permanently deletes a company.
const CompanyDeletedLabel Label = "company.deleted"

// CompanySetAsDefaultLabel labels the event dispatched when a user makes a company their default.
const CompanySetAsDefaultLabel Label = "company.set_as_default"

// InvoiceTemplateCreatedLabel labels the event dispatched when a user records an invoice template.
const InvoiceTemplateCreatedLabel Label = "invoice_template.created"

// InvoiceTemplateUpdatedLabel labels the event dispatched when a user replaces an invoice template's name and layout.
const InvoiceTemplateUpdatedLabel Label = "invoice_template.updated"

// InvoiceTemplateDeletedLabel labels the event dispatched when a user permanently deletes an invoice template.
const InvoiceTemplateDeletedLabel Label = "invoice_template.deleted"

// InvoiceTemplateSetAsDefaultLabel labels the event dispatched when a user makes an invoice template their default.
const InvoiceTemplateSetAsDefaultLabel Label = "invoice_template.set_as_default"

// ClientCreatedLabel labels the event dispatched when a user records a client they invoice.
const ClientCreatedLabel Label = "client.created"

// ClientUpdatedLabel labels the event dispatched when a user replaces the details of a client they invoice.
const ClientUpdatedLabel Label = "client.updated"

// ClientDeletedLabel labels the event dispatched when a user permanently deletes a client they invoice.
const ClientDeletedLabel Label = "client.deleted"
