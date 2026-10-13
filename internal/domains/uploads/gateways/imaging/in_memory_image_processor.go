package imaging

import (
	"context"
	"fmt"
)

var _ ImageProcessor = (*InMemoryImageProcessor)(nil)

// InMemoryImageProcessor is an ImageProcessor that reports staged dimensions and fakes previews.
type InMemoryImageProcessor struct {
	// Dimensions is what every inspected image reports.
	Dimensions ImageInfo
	Previewed  []CreatePreviewParams

	// InspectFailure and PreviewFailure, when set, are returned instead of doing the work.
	InspectFailure error
	PreviewFailure error
}

// NewInMemoryImageProcessor builds a processor reporting every image as the given dimensions.
func NewInMemoryImageProcessor(dimensions ImageInfo) *InMemoryImageProcessor {
	return &InMemoryImageProcessor{Dimensions: dimensions}
}

// InspectImage reports the staged dimensions, or InspectFailure when one is staged.
func (p *InMemoryImageProcessor) InspectImage(_ context.Context, _ []byte) (ImageInfo, error) {
	if p.InspectFailure != nil {
		return ImageInfo{}, p.InspectFailure
	}

	return p.Dimensions, nil
}

// CreatePreview records the request and answers with InMemoryPreviewContent of its edge.
func (p *InMemoryImageProcessor) CreatePreview(_ context.Context, params CreatePreviewParams) ([]byte, error) {
	if p.PreviewFailure != nil {
		return nil, p.PreviewFailure
	}

	p.Previewed = append(p.Previewed, params)

	return InMemoryPreviewContent(params.EdgePixels), nil
}

// InMemoryPreviewContent is the placeholder preview the in-memory processor produces for an edge.
func InMemoryPreviewContent(edgePixels int) []byte {
	return []byte(fmt.Sprintf("preview-%dpx", edgePixels))
}
