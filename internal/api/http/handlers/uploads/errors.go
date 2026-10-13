package uploads

import "fmt"

const unexpectedUploadFailureText = "could not upload the files"

const uploadRejectedText = "rejected a file upload"

const uploadFormCleanupFailedText = "failed removing the temp files of a parsed upload form"

const uploadFormUndecodableText = "could not read the multipart form"

// ErrUploadContentTypeUnsupported reports an upload whose Content-Type is not multipart/form-data.
type ErrUploadContentTypeUnsupported struct {
	ContentType string
}

func (e ErrUploadContentTypeUnsupported) Error() string {
	return fmt.Sprintf("unsupported Content-Type %q: expected %s", e.ContentType, multipartMediaType)
}

// ErrUploadFormUndecodable reports a multipart body that is malformed or cut short.
type ErrUploadFormUndecodable struct {
	Cause error
}

func (e ErrUploadFormUndecodable) Error() string {
	return fmt.Sprintf("%s: %s", uploadFormUndecodableText, e.Cause)
}

func (e ErrUploadFormUndecodable) Unwrap() error {
	return e.Cause
}
