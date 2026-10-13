package firebase

import "fmt"

const bucketNameMissingText = "expected the name of the storage bucket files are kept in"
const credentialsUndecodableText = "expected the service account JSON encoded as standard base64"
const clientUnavailableText = "failed building the storage client from the service account"
const objectWriteFailedText = "failed writing the object"
const objectDeleteFailedText = "failed deleting the object"

// ErrBucketNameMissing reports a gateway built without a bucket name.
type ErrBucketNameMissing struct{}

func (e ErrBucketNameMissing) Error() string {
	return fmt.Sprintf("missing bucket name: %s", bucketNameMissingText)
}

// ErrCredentialsUndecodable reports service account credentials that are not valid base64.
type ErrCredentialsUndecodable struct {
	Cause error
}

func (e ErrCredentialsUndecodable) Error() string {
	return fmt.Sprintf("invalid credentials: %s: %s", credentialsUndecodableText, e.Cause)
}

func (e ErrCredentialsUndecodable) Unwrap() error {
	return e.Cause
}

// ErrClientUnavailable reports a storage client the decoded service account could not build.
type ErrClientUnavailable struct {
	Cause error
}

func (e ErrClientUnavailable) Error() string {
	return fmt.Sprintf("%s: %s", clientUnavailableText, e.Cause)
}

func (e ErrClientUnavailable) Unwrap() error {
	return e.Cause
}

// ErrObjectWriteFailed reports an object the bucket did not accept.
type ErrObjectWriteFailed struct {
	Path  string
	Cause error
}

func (e ErrObjectWriteFailed) Error() string {
	return fmt.Sprintf("%s %q: %s", objectWriteFailedText, e.Path, e.Cause)
}

func (e ErrObjectWriteFailed) Unwrap() error {
	return e.Cause
}

// ErrObjectDeleteFailed reports an object the bucket failed to delete.
type ErrObjectDeleteFailed struct {
	Path  string
	Cause error
}

func (e ErrObjectDeleteFailed) Error() string {
	return fmt.Sprintf("%s %q: %s", objectDeleteFailedText, e.Path, e.Cause)
}

func (e ErrObjectDeleteFailed) Unwrap() error {
	return e.Cause
}
