package models

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Purpose is what an upload will be attached to, deciding which files it accepts and its preview.
type Purpose string

const (
	PurposeProfilePicture     Purpose = "profile_picture"
	PurposeDocumentAttachment Purpose = "document_attachment"
)

// Kind is the broad family of a file, deciding which checks and processing it gets.
type Kind string

const (
	KindImage    Kind = "image"
	KindDocument Kind = "document"
)

const maxFileSize = 5 << 20

// MaxFilesPerUpload is the most files one upload request may carry, whatever its purpose.
const MaxFilesPerUpload = 10

// MaxImageEdgePixels bounds image width and height, so a small file cannot decode into a huge bitmap.
const MaxImageEdgePixels = 10000

// PendingUploadRetention is how long an unattached upload is kept before cleanup removes it.
const PendingUploadRetention = 24 * time.Hour

// PreviewMimeType is the format every image preview is encoded in.
const PreviewMimeType = "image/webp"

const previewFileName = "preview.webp"

// FileFormat is a file format the server accepts, as detected from the file's content.
type FileFormat struct {
	MimeType     string
	Kind         Kind
	Extension    string
	MaxSizeBytes int64
}

// The keys are the names http.DetectContentType reports, so they must stay in its spelling.
var fileFormatsByMimeType = map[string]FileFormat{
	"image/jpeg":      {MimeType: "image/jpeg", Kind: KindImage, Extension: "jpg", MaxSizeBytes: maxFileSize},
	"image/png":       {MimeType: "image/png", Kind: KindImage, Extension: "png", MaxSizeBytes: maxFileSize},
	"image/webp":      {MimeType: "image/webp", Kind: KindImage, Extension: "webp", MaxSizeBytes: maxFileSize},
	"application/pdf": {MimeType: "application/pdf", Kind: KindDocument, Extension: "pdf", MaxSizeBytes: maxFileSize},
}

// DetectFileFormat sniffs content for an accepted format, returning the detected MIME type either way.
func DetectFileFormat(content []byte) (FileFormat, string, bool) {
	detectedMimeType := http.DetectContentType(content)

	format, accepted := fileFormatsByMimeType[detectedMimeType]

	return format, detectedMimeType, accepted
}

// PreviewSpec is how an image preview is sized: fit inside an Edge-pixel square, or cropped to fill it.
type PreviewSpec struct {
	EdgePixels int
	Crop       bool
}

// PurposeRules is what an upload purpose accepts, how many files per request, and how its image
// previews are made.
type PurposeRules struct {
	AllowedKinds []Kind
	MaxFiles     int
	Preview      PreviewSpec
}

var purposeRulesByPurpose = map[Purpose]PurposeRules{
	PurposeProfilePicture: {
		AllowedKinds: []Kind{KindImage},
		MaxFiles:     1,
		Preview:      PreviewSpec{EdgePixels: 512, Crop: true},
	},
	PurposeDocumentAttachment: {
		AllowedKinds: []Kind{KindImage, KindDocument},
		MaxFiles:     MaxFilesPerUpload,
		Preview:      PreviewSpec{EdgePixels: 1024, Crop: false},
	},
}

// RulesForPurpose returns the rules of a known purpose, and false for an unknown one.
func RulesForPurpose(purpose Purpose) (PurposeRules, bool) {
	rules, known := purposeRulesByPurpose[purpose]

	return rules, known
}

// AllowsKind reports whether the purpose accepts files of the kind.
func (r PurposeRules) AllowsKind(kind Kind) bool {
	return slices.Contains(r.AllowedKinds, kind)
}

// StoragePathsParams names the upload whose file storage paths are built.
type StoragePathsParams struct {
	AppEnv    string
	UserID    uuid.UUID
	Purpose   Purpose
	UploadID  uuid.UUID
	Extension string
}

// StoragePaths is where an upload's original file and its preview live in file storage.
type StoragePaths struct {
	Original string
	Preview  string
}

// BuildStoragePaths places an upload under its environment, owner, purpose and id.
func BuildStoragePaths(params StoragePathsParams) StoragePaths {
	directory := fmt.Sprintf("uploads/%s/users/%s/%s/%s",
		params.AppEnv, params.UserID, params.Purpose, params.UploadID)

	return StoragePaths{
		Original: fmt.Sprintf("%s/original.%s", directory, params.Extension),
		Preview:  fmt.Sprintf("%s/%s", directory, previewFileName),
	}
}
