// Package errors holds the failures the upload use cases, ports and adapters report.
package errors

import (
	"fmt"

	"github.com/google/uuid"
)

const uploadPurposeUnknownText = "expected one of the known upload purposes"
const uploadFilesMissingText = "expected at least one file"
const tooManyUploadFilesText = "expected no more files than the upload purpose accepts per request"
const uploadFileTypeUnsupportedText = "expected a jpeg, png, webp or pdf file"
const uploadFileTypeNotAllowedText = "expected a file kind the upload purpose accepts"
const uploadFileTooLargeText = "expected a file no larger than the limit for its type"
const imageDimensionsTooLargeText = "expected an image no wider or taller than the limit"
const imageUnreadableText = "failed reading the image, it may be corrupted"
const createPreviewFailedText = "failed creating the image preview"
const storeFileFailedText = "failed storing the file"
const storedFileNotFoundText = "stored file not found"
const deleteStoredFileFailedText = "failed deleting the stored file"
const saveUploadFailedText = "failed saving the upload"
const listPendingUploadsFailedText = "failed listing the pending uploads"
const deleteUploadFailedText = "failed deleting the upload"

// DiscardStoredFileFailedText is logged when a file stored for a failed upload could not be removed.
const DiscardStoredFileFailedText = "failed discarding a file stored for a failed upload"

// ErrUploadPurposeUnknown reports an upload asked for with a purpose the domain does not know.
type ErrUploadPurposeUnknown struct {
	Purpose string
}

func (e ErrUploadPurposeUnknown) Error() string {
	return fmt.Sprintf("upload purpose %q: %s", e.Purpose, uploadPurposeUnknownText)
}

// ErrUploadFilesMissing reports an upload asked for without any file.
type ErrUploadFilesMissing struct{}

func (e ErrUploadFilesMissing) Error() string {
	return fmt.Sprintf("no files: %s", uploadFilesMissingText)
}

// ErrTooManyUploadFiles reports an upload carrying more files than its purpose accepts in one request.
type ErrTooManyUploadFiles struct {
	Count int
	Limit int
}

func (e ErrTooManyUploadFiles) Error() string {
	return fmt.Sprintf("%d files: %s of %d", e.Count, tooManyUploadFilesText, e.Limit)
}

// ErrUploadFileTypeUnsupported reports a file whose content is not a format the server accepts.
type ErrUploadFileTypeUnsupported struct {
	FileName         string
	DetectedMimeType string
}

func (e ErrUploadFileTypeUnsupported) Error() string {
	return fmt.Sprintf("file %q is %s: %s", e.FileName, e.DetectedMimeType, uploadFileTypeUnsupportedText)
}

// ErrUploadFileTypeNotAllowed reports a supported file whose kind the upload purpose does not accept.
type ErrUploadFileTypeNotAllowed struct {
	FileName string
	Kind     string
	Purpose  string
}

func (e ErrUploadFileTypeNotAllowed) Error() string {
	return fmt.Sprintf("file %q is a %s, not accepted as %s: %s",
		e.FileName, e.Kind, e.Purpose, uploadFileTypeNotAllowedText)
}

// ErrUploadFileTooLarge reports a file over the size limit of its format.
type ErrUploadFileTooLarge struct {
	FileName     string
	SizeBytes    int64
	MaxSizeBytes int64
}

func (e ErrUploadFileTooLarge) Error() string {
	return fmt.Sprintf("file %q is %d bytes: %s of %d bytes",
		e.FileName, e.SizeBytes, uploadFileTooLargeText, e.MaxSizeBytes)
}

// ErrImageDimensionsTooLarge reports an image wider or taller than the pixel limit.
type ErrImageDimensionsTooLarge struct {
	FileName      string
	Width         int
	Height        int
	MaxEdgePixels int
}

func (e ErrImageDimensionsTooLarge) Error() string {
	return fmt.Sprintf("image %q is %dx%d px: %s of %d px",
		e.FileName, e.Width, e.Height, imageDimensionsTooLargeText, e.MaxEdgePixels)
}

// ErrImageUnreadable reports an image whose bytes the image processor could not decode.
type ErrImageUnreadable struct {
	FileName string
	Cause    error
}

func (e ErrImageUnreadable) Error() string {
	return fmt.Sprintf("image %q: %s: %s", e.FileName, imageUnreadableText, e.Cause)
}

func (e ErrImageUnreadable) Unwrap() error {
	return e.Cause
}

// ErrCreatePreviewFailed reports an image the image processor failed to make a preview of.
type ErrCreatePreviewFailed struct {
	FileName string
	Cause    error
}

func (e ErrCreatePreviewFailed) Error() string {
	return fmt.Sprintf("%s for %q: %s", createPreviewFailedText, e.FileName, e.Cause)
}

func (e ErrCreatePreviewFailed) Unwrap() error {
	return e.Cause
}

// ErrStoreFileFailed reports a file the file storage refused to store.
type ErrStoreFileFailed struct {
	Path  string
	Cause error
}

func (e ErrStoreFileFailed) Error() string {
	return fmt.Sprintf("%s at %q: %s", storeFileFailedText, e.Path, e.Cause)
}

func (e ErrStoreFileFailed) Unwrap() error {
	return e.Cause
}

// ErrStoredFileNotFound reports a delete of a path the file storage holds nothing at.
type ErrStoredFileNotFound struct {
	Path string
}

func (e ErrStoredFileNotFound) Error() string {
	return fmt.Sprintf("%s at %q", storedFileNotFoundText, e.Path)
}

// ErrDeleteStoredFileFailed reports a stored file the file storage failed to delete.
type ErrDeleteStoredFileFailed struct {
	Path  string
	Cause error
}

func (e ErrDeleteStoredFileFailed) Error() string {
	return fmt.Sprintf("%s at %q: %s", deleteStoredFileFailedText, e.Path, e.Cause)
}

func (e ErrDeleteStoredFileFailed) Unwrap() error {
	return e.Cause
}

// ErrSaveUploadFailed reports an upload row the database failed to insert.
type ErrSaveUploadFailed struct {
	ID    uuid.UUID
	Cause error
}

func (e ErrSaveUploadFailed) Error() string {
	return fmt.Sprintf("%s %s: %s", saveUploadFailedText, e.ID, e.Cause)
}

func (e ErrSaveUploadFailed) Unwrap() error {
	return e.Cause
}

// ErrListPendingUploadsFailed reports a pending upload lookup the database could not complete.
type ErrListPendingUploadsFailed struct {
	Cause error
}

func (e ErrListPendingUploadsFailed) Error() string {
	return fmt.Sprintf("%s: %s", listPendingUploadsFailedText, e.Cause)
}

func (e ErrListPendingUploadsFailed) Unwrap() error {
	return e.Cause
}

// ErrDeleteUploadFailed reports an upload row the database failed to delete.
type ErrDeleteUploadFailed struct {
	ID    uuid.UUID
	Cause error
}

func (e ErrDeleteUploadFailed) Error() string {
	return fmt.Sprintf("%s %s: %s", deleteUploadFailedText, e.ID, e.Cause)
}

func (e ErrDeleteUploadFailed) Unwrap() error {
	return e.Cause
}
