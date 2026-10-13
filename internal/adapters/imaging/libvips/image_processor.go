// Package libvips implements the image processor port with libvips, through govips.
package libvips

import (
	"context"
	"log/slog"

	"github.com/davidbyttow/govips/v2/vips"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
)

// previewWebpQuality trades a little visible detail for previews a fraction of the original's size.
const previewWebpQuality = 80

var _ imaging.ImageProcessor = ImageProcessor{}

// ImageProcessor inspects and compresses images with libvips; it is safe for concurrent use.
type ImageProcessor struct{}

// NewImageProcessor starts libvips once per process; call Shutdown when the process stops.
func NewImageProcessor() (ImageProcessor, error) {
	vips.LoggingSettings(forwardLibvipsLog, vips.LogLevelWarning)

	if err := vips.Startup(nil); err != nil {
		return ImageProcessor{}, ErrLibvipsStartupFailed{Cause: err}
	}

	return ImageProcessor{}, nil
}

// Shutdown releases libvips; libvips cannot be started again in the same process afterwards.
func (ImageProcessor) Shutdown() {
	vips.Shutdown()
}

// InspectImage reads the image header for its dimensions without decoding the pixels.
func (ImageProcessor) InspectImage(_ context.Context, content []byte) (imaging.ImageInfo, error) {
	image, err := vips.NewImageFromBuffer(content)
	if err != nil {
		return imaging.ImageInfo{}, ErrImageDecodeFailed{Cause: err}
	}
	defer image.Close()

	return imaging.ImageInfo{Width: image.Width(), Height: image.Height()}, nil
}

// CreatePreview shrinks the image, upright per its EXIF orientation, and encodes it as WebP
// without metadata. Images already inside the square are never enlarged.
func (ImageProcessor) CreatePreview(_ context.Context, params imaging.CreatePreviewParams) ([]byte, error) {
	crop := vips.InterestingNone
	if params.Crop {
		crop = vips.InterestingCentre
	}

	thumbnail, err := vips.NewThumbnailWithSizeFromBuffer(
		params.Content, params.EdgePixels, params.EdgePixels, crop, vips.SizeDown)
	if err != nil {
		return nil, ErrThumbnailFailed{EdgePixels: params.EdgePixels, Cause: err}
	}
	defer thumbnail.Close()

	exportParams := vips.NewWebpExportParams()
	exportParams.Quality = previewWebpQuality
	exportParams.StripMetadata = true

	encoded, _, err := thumbnail.ExportWebp(exportParams)
	if err != nil {
		return nil, ErrWebpExportFailed{Cause: err}
	}

	return encoded, nil
}

func forwardLibvipsLog(messageDomain string, messageLevel vips.LogLevel, message string) {
	slog.Warn("libvips", "domain", messageDomain, "level", int(messageLevel), "message", message)
}
