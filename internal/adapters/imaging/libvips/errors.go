package libvips

import "fmt"

const libvipsStartupFailedText = "failed starting libvips"
const imageDecodeFailedText = "failed decoding the image"
const thumbnailFailedText = "failed shrinking the image"
const webpExportFailedText = "failed encoding the preview as webp"

// ErrLibvipsStartupFailed reports a libvips that could not be initialised.
type ErrLibvipsStartupFailed struct {
	Cause error
}

func (e ErrLibvipsStartupFailed) Error() string {
	return fmt.Sprintf("%s: %s", libvipsStartupFailedText, e.Cause)
}

func (e ErrLibvipsStartupFailed) Unwrap() error {
	return e.Cause
}

// ErrImageDecodeFailed reports image bytes libvips could not read.
type ErrImageDecodeFailed struct {
	Cause error
}

func (e ErrImageDecodeFailed) Error() string {
	return fmt.Sprintf("%s: %s", imageDecodeFailedText, e.Cause)
}

func (e ErrImageDecodeFailed) Unwrap() error {
	return e.Cause
}

// ErrThumbnailFailed reports an image libvips could not shrink to the preview square.
type ErrThumbnailFailed struct {
	EdgePixels int
	Cause      error
}

func (e ErrThumbnailFailed) Error() string {
	return fmt.Sprintf("%s to %dpx: %s", thumbnailFailedText, e.EdgePixels, e.Cause)
}

func (e ErrThumbnailFailed) Unwrap() error {
	return e.Cause
}

// ErrWebpExportFailed reports a shrunk image libvips could not encode as WebP.
type ErrWebpExportFailed struct {
	Cause error
}

func (e ErrWebpExportFailed) Error() string {
	return fmt.Sprintf("%s: %s", webpExportFailedText, e.Cause)
}

func (e ErrWebpExportFailed) Unwrap() error {
	return e.Cause
}
