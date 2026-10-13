package libvips

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"golang.org/x/image/webp"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
)

var processor ImageProcessor

func TestMain(m *testing.M) {
	started, err := NewImageProcessor()
	if err != nil {
		panic(err)
	}
	processor = started

	exitCode := m.Run()

	processor.Shutdown()
	os.Exit(exitCode)
}

func encodedPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			canvas.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatalf("encoding the fixture png: %v", err)
	}

	return encoded.Bytes()
}

func decodedWebpBounds(t *testing.T, content []byte) image.Rectangle {
	t.Helper()

	decoded, err := webp.Decode(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("the preview is not a decodable webp: %v", err)
	}

	return decoded.Bounds()
}

func TestInspectImageReportsTheImageDimensions(t *testing.T) {
	info, err := processor.InspectImage(context.Background(), encodedPNG(t, 300, 200))
	if err != nil {
		t.Fatalf("InspectImage: %v", err)
	}

	if info != (imaging.ImageInfo{Width: 300, Height: 200}) {
		t.Errorf("info = %+v, want 300x200", info)
	}
}

func TestInspectImageRejectsBytesThatAreNotAnImage(t *testing.T) {
	_, err := processor.InspectImage(context.Background(), []byte("\x89PNG\r\n\x1a\n truncated"))

	if _, ok := errors.AsType[ErrImageDecodeFailed](err); !ok {
		t.Fatalf("err = %v, want ErrImageDecodeFailed", err)
	}
}

func TestCreatePreviewShrinksIntoTheSquareAsWebp(t *testing.T) {
	cases := []struct {
		name       string
		width      int
		height     int
		edgePixels int
		crop       bool
		wantWidth  int
		wantHeight int
	}{
		{name: "fit keeps the aspect ratio", width: 400, height: 200, edgePixels: 100, wantWidth: 100, wantHeight: 50},
		{name: "crop fills the square", width: 400, height: 200, edgePixels: 100, crop: true, wantWidth: 100, wantHeight: 100},
		{name: "small image is not enlarged", width: 60, height: 30, edgePixels: 100, wantWidth: 60, wantHeight: 30},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			preview, err := processor.CreatePreview(context.Background(), imaging.CreatePreviewParams{
				Content:    encodedPNG(t, testCase.width, testCase.height),
				EdgePixels: testCase.edgePixels,
				Crop:       testCase.crop,
			})
			if err != nil {
				t.Fatalf("CreatePreview: %v", err)
			}

			bounds := decodedWebpBounds(t, preview)
			if bounds.Dx() != testCase.wantWidth || bounds.Dy() != testCase.wantHeight {
				t.Errorf("preview = %dx%d, want %dx%d",
					bounds.Dx(), bounds.Dy(), testCase.wantWidth, testCase.wantHeight)
			}
		})
	}
}

func TestCreatePreviewRejectsBytesThatAreNotAnImage(t *testing.T) {
	_, err := processor.CreatePreview(context.Background(), imaging.CreatePreviewParams{
		Content:    []byte("not an image"),
		EdgePixels: 100,
	})

	if _, ok := errors.AsType[ErrThumbnailFailed](err); !ok {
		t.Fatalf("err = %v, want ErrThumbnailFailed", err)
	}
}
