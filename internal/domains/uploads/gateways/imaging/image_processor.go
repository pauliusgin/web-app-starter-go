// Package imaging declares the outbound port images are inspected and compressed through.
package imaging

import "context"

// ImageInfo is what inspecting an image reveals without decoding its pixels.
type ImageInfo struct {
	Width  int
	Height int
}

// CreatePreviewParams is the image to compress and the square it must fit in, or fill when Crop is set.
type CreatePreviewParams struct {
	Content    []byte
	EdgePixels int
	Crop       bool
}

// ImageProcessor is the outbound image port the domain calls and an imaging library adapter implements.
type ImageProcessor interface {
	InspectImage(ctx context.Context, content []byte) (ImageInfo, error)

	// CreatePreview returns the image upright, shrunk to the requested square and encoded as WebP.
	CreatePreview(ctx context.Context, params CreatePreviewParams) ([]byte, error)
}
