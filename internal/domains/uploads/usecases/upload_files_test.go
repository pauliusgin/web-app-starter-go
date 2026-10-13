package usecases

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testAppEnv = "test"

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var jpegContent = []byte("\xFF\xD8\xFF\xE0 jpeg bytes")

var pdfContent = []byte("%PDF-1.7\n pdf bytes")

var errStorageDown = errors.New("storage unreachable")

var errDatabaseDown = errors.New("database unreachable")

type capturedEvents struct {
	recorded  []events.NewEvent
	published []events.NewEvent
}

func (c *capturedEvents) dispatcher() events.Dispatcher {
	return events.Dispatcher{
		Recorder: func(_ context.Context, event events.NewEvent) error {
			c.recorded = append(c.recorded, event)
			return nil
		},
		Publisher: func(_ context.Context, event events.NewEvent) error {
			c.published = append(c.published, event)
			return nil
		},
	}
}

// uploadFilesFixture wires the use case to in-memory collaborators it can be inspected through.
type uploadFilesFixture struct {
	repository     *repositories.InMemoryUploadRepository
	fileStorage    *storage.InMemoryFileStorage
	imageProcessor *imaging.InMemoryImageProcessor
	captured       *capturedEvents
}

func newUploadFilesFixture() uploadFilesFixture {
	return uploadFilesFixture{
		repository:     repositories.NewInMemoryUploadRepository(),
		fileStorage:    storage.NewInMemoryFileStorage(),
		imageProcessor: imaging.NewInMemoryImageProcessor(imaging.ImageInfo{Width: 1600, Height: 1200}),
		captured:       &capturedEvents{},
	}
}

func (f uploadFilesFixture) upload(params UploadFilesParams) (UploadFilesResult, error) {
	return UploadFiles(context.Background(), UploadFilesDependencies{
		UploadRepository: f.repository,
		FileStorage:      f.fileStorage,
		ImageProcessor:   f.imageProcessor,
		EventDispatcher:  f.captured.dispatcher(),
		AppEnv:           testAppEnv,
	}, params)
}

func uploadParams(purpose models.Purpose, files ...IncomingFile) UploadFilesParams {
	return UploadFilesParams{UserID: testUserID, Purpose: purpose, Files: files}
}

func jpegFile(name string) IncomingFile {
	return IncomingFile{Name: name, Content: jpegContent}
}

func pdfFile(name string) IncomingFile {
	return IncomingFile{Name: name, Content: pdfContent}
}

// assertNothingUploaded fails the test if the rejected upload left any file, row or event behind.
func assertNothingUploaded(t *testing.T, f uploadFilesFixture, uploads []models.Upload) {
	t.Helper()

	if uploads != nil {
		t.Errorf("uploads = %+v, want nil", uploads)
	}
	if len(f.fileStorage.Files) != 0 {
		t.Errorf("stored files = %v, want none", f.fileStorage.Files)
	}
	if len(f.repository.Uploads) != 0 {
		t.Errorf("saved uploads = %+v, want none", f.repository.Uploads)
	}
	if len(f.captured.recorded) != 0 || len(f.captured.published) != 0 {
		t.Errorf("recorded = %d, published = %d, want no events", len(f.captured.recorded), len(f.captured.published))
	}
}

func TestUploadFilesStoresAnImageWithItsPreviewAndRecordsItPending(t *testing.T) {
	fixture := newUploadFilesFixture()

	result, err := fixture.upload(uploadParams(models.PurposeProfilePicture, jpegFile("me.jpg")))
	if err != nil {
		t.Fatalf("UploadFiles: %v", err)
	}
	if len(result.Uploads) != 1 {
		t.Fatalf("uploads = %d, want 1", len(result.Uploads))
	}

	upload := result.Uploads[0]
	paths := models.BuildStoragePaths(models.StoragePathsParams{
		AppEnv:    testAppEnv,
		UserID:    testUserID,
		Purpose:   models.PurposeProfilePicture,
		UploadID:  upload.ID,
		Extension: "jpg",
	})
	want := models.Upload{
		ID:           upload.ID,
		UserID:       testUserID,
		Purpose:      models.PurposeProfilePicture,
		Kind:         models.KindImage,
		MimeType:     "image/jpeg",
		SizeBytes:    int64(len(jpegContent)),
		Width:        1600,
		Height:       1200,
		OriginalPath: paths.Original,
		OriginalURL:  storage.InMemoryFileURL(paths.Original),
		PreviewPath:  paths.Preview,
		PreviewURL:   storage.InMemoryFileURL(paths.Preview),
		CreatedAt:    upload.CreatedAt,
		UpdatedAt:    upload.CreatedAt,
	}
	if upload != want {
		t.Errorf("upload = %+v\nwant     %+v", upload, want)
	}
	if upload.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want the save time")
	}
	if upload.IsAttached() {
		t.Error("a fresh upload is attached, want pending")
	}

	if len(fixture.repository.Uploads) != 1 || fixture.repository.Uploads[0] != want {
		t.Errorf("saved uploads = %+v, want only the returned upload", fixture.repository.Uploads)
	}

	original := fixture.fileStorage.Files[paths.Original]
	if original.ContentType != "image/jpeg" || !bytes.Equal(original.Content, jpegContent) {
		t.Errorf("stored original = %+v, want the uploaded jpeg bytes", original)
	}
	preview := fixture.fileStorage.Files[paths.Preview]
	if preview.ContentType != models.PreviewMimeType || !bytes.Equal(preview.Content, imaging.InMemoryPreviewContent(512)) {
		t.Errorf("stored preview = %+v, want the 512px webp preview", preview)
	}

	wantPreview := imaging.CreatePreviewParams{Content: jpegContent, EdgePixels: 512, Crop: true}
	if len(fixture.imageProcessor.Previewed) != 1 || !samePreviewRequest(fixture.imageProcessor.Previewed[0], wantPreview) {
		t.Errorf("preview requests = %+v, want one square crop at 512px", fixture.imageProcessor.Previewed)
	}
}

func samePreviewRequest(got, want imaging.CreatePreviewParams) bool {
	return bytes.Equal(got.Content, want.Content) && got.EdgePixels == want.EdgePixels && got.Crop == want.Crop
}

func TestUploadFilesStoresADocumentWithoutAPreview(t *testing.T) {
	fixture := newUploadFilesFixture()

	result, err := fixture.upload(uploadParams(models.PurposeDocumentAttachment, pdfFile("document.pdf")))
	if err != nil {
		t.Fatalf("UploadFiles: %v", err)
	}

	upload := result.Uploads[0]
	if upload.Kind != models.KindDocument || upload.MimeType != "application/pdf" {
		t.Errorf("upload kind = %q mime = %q, want a pdf document", upload.Kind, upload.MimeType)
	}
	if upload.PreviewPath != "" || upload.PreviewURL != "" {
		t.Errorf("preview = %q %q, want none for a document", upload.PreviewPath, upload.PreviewURL)
	}
	if upload.Width != 0 || upload.Height != 0 {
		t.Errorf("dimensions = %dx%d, want none for a document", upload.Width, upload.Height)
	}
	if len(fixture.fileStorage.Files) != 1 {
		t.Errorf("stored files = %d, want only the original", len(fixture.fileStorage.Files))
	}
	if len(fixture.imageProcessor.Previewed) != 0 {
		t.Errorf("preview requests = %d, want none for a document", len(fixture.imageProcessor.Previewed))
	}
}

func TestUploadFilesStoresEveryFileInRequestOrder(t *testing.T) {
	fixture := newUploadFilesFixture()

	result, err := fixture.upload(uploadParams(models.PurposeDocumentAttachment,
		jpegFile("scan.jpg"), pdfFile("document.pdf")))
	if err != nil {
		t.Fatalf("UploadFiles: %v", err)
	}

	if len(result.Uploads) != 2 || result.Uploads[0].Kind != models.KindImage || result.Uploads[1].Kind != models.KindDocument {
		t.Fatalf("uploads = %+v, want the image then the document", result.Uploads)
	}
	if result.Uploads[0].ID == result.Uploads[1].ID {
		t.Error("both uploads share an id, want one id each")
	}
	if len(fixture.repository.Uploads) != 2 {
		t.Errorf("saved uploads = %d, want 2", len(fixture.repository.Uploads))
	}
	if len(fixture.fileStorage.Files) != 3 {
		t.Errorf("stored files = %d, want image original + preview + pdf", len(fixture.fileStorage.Files))
	}
}

func TestUploadFilesAnnouncesEachStoredUpload(t *testing.T) {
	fixture := newUploadFilesFixture()

	result, err := fixture.upload(uploadParams(models.PurposeDocumentAttachment,
		jpegFile("scan.jpg"), pdfFile("document.pdf")))
	if err != nil {
		t.Fatalf("UploadFiles: %v", err)
	}

	if len(fixture.captured.recorded) != 2 || len(fixture.captured.published) != 2 {
		t.Fatalf("recorded = %d, published = %d, want 2 each",
			len(fixture.captured.recorded), len(fixture.captured.published))
	}
	for index, event := range fixture.captured.published {
		if event.Label != events.UploadCreatedLabel {
			t.Errorf("event %d label = %q, want %q", index, event.Label, events.UploadCreatedLabel)
		}
		if event.UserID != (uuid.NullUUID{UUID: testUserID, Valid: true}) {
			t.Errorf("event %d user = %+v, want %s", index, event.UserID, testUserID)
		}
		if !bytes.Contains(event.Data, []byte(result.Uploads[index].ID.String())) {
			t.Errorf("event %d payload %s, want upload id %s", index, event.Data, result.Uploads[index].ID)
		}
	}
}

func TestUploadFilesSucceedsWhenTheEventCannotBeDispatched(t *testing.T) {
	fixture := newUploadFilesFixture()
	failing := func(context.Context, events.NewEvent) error { return errors.New("broker down") }

	result, err := UploadFiles(context.Background(), UploadFilesDependencies{
		UploadRepository: fixture.repository,
		FileStorage:      fixture.fileStorage,
		ImageProcessor:   fixture.imageProcessor,
		EventDispatcher:  events.Dispatcher{Recorder: failing, Publisher: failing},
		AppEnv:           testAppEnv,
	}, uploadParams(models.PurposeProfilePicture, jpegFile("me.jpg")))
	if err != nil {
		t.Fatalf("UploadFiles: %v, want success despite the failing dispatcher", err)
	}

	if len(result.Uploads) != 1 || len(fixture.repository.Uploads) != 1 {
		t.Errorf("uploads = %d, saved = %d, want 1 each", len(result.Uploads), len(fixture.repository.Uploads))
	}
}

func TestUploadFilesRejectsABadRequestBeforeStoringAnything(t *testing.T) {
	oversizedJPEG := append(append([]byte{}, jpegContent...), make([]byte, 5<<20)...)
	elevenFiles := make([]IncomingFile, models.MaxFilesPerUpload+1)
	for index := range elevenFiles {
		elevenFiles[index] = jpegFile("photo.jpg")
	}

	cases := []struct {
		name    string
		params  UploadFilesParams
		wantErr func(err error) bool
	}{
		{
			name:    "unknown purpose",
			params:  uploadParams("cover_photo", jpegFile("me.jpg")),
			wantErr: errorOfType[e.ErrUploadPurposeUnknown](),
		},
		{
			name:    "no files",
			params:  uploadParams(models.PurposeProfilePicture),
			wantErr: errorOfType[e.ErrUploadFilesMissing](),
		},
		{
			name:    "too many files",
			params:  uploadParams(models.PurposeDocumentAttachment, elevenFiles...),
			wantErr: errorOfType[e.ErrTooManyUploadFiles](),
		},
		{
			name:    "two profile pictures",
			params:  uploadParams(models.PurposeProfilePicture, jpegFile("me.jpg"), jpegFile("me-too.jpg")),
			wantErr: errorOfType[e.ErrTooManyUploadFiles](),
		},
		{
			name:    "unsupported format",
			params:  uploadParams(models.PurposeDocumentAttachment, IncomingFile{Name: "notes.txt", Content: []byte("notes")}),
			wantErr: errorOfType[e.ErrUploadFileTypeUnsupported](),
		},
		{
			name:    "document as profile picture",
			params:  uploadParams(models.PurposeProfilePicture, pdfFile("cv.pdf")),
			wantErr: errorOfType[e.ErrUploadFileTypeNotAllowed](),
		},
		{
			name:    "oversized image",
			params:  uploadParams(models.PurposeProfilePicture, IncomingFile{Name: "huge.jpg", Content: oversizedJPEG}),
			wantErr: errorOfType[e.ErrUploadFileTooLarge](),
		},
		{
			name: "valid file followed by an invalid one",
			params: uploadParams(models.PurposeDocumentAttachment,
				jpegFile("scan.jpg"), IncomingFile{Name: "notes.txt", Content: []byte("notes")}),
			wantErr: errorOfType[e.ErrUploadFileTypeUnsupported](),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newUploadFilesFixture()

			result, err := fixture.upload(testCase.params)

			if !testCase.wantErr(err) {
				t.Fatalf("err = %v (%T), want the rejection this case names", err, err)
			}
			assertNothingUploaded(t, fixture, result.Uploads)
		})
	}
}

// errorOfType builds a check that an error's chain holds an E.
func errorOfType[E error]() func(err error) bool {
	return func(err error) bool {
		_, ok := errors.AsType[E](err)
		return ok
	}
}

func TestUploadFilesRejectsAnImageOverTheDimensionLimit(t *testing.T) {
	fixture := newUploadFilesFixture()
	fixture.imageProcessor.Dimensions = imaging.ImageInfo{Width: models.MaxImageEdgePixels + 1, Height: 100}

	result, err := fixture.upload(uploadParams(models.PurposeProfilePicture, jpegFile("panorama.jpg")))

	tooLarge, ok := errors.AsType[e.ErrImageDimensionsTooLarge](err)
	if !ok {
		t.Fatalf("err = %v, want ErrImageDimensionsTooLarge", err)
	}
	if tooLarge.FileName != "panorama.jpg" || tooLarge.Width != models.MaxImageEdgePixels+1 {
		t.Errorf("err = %+v, want the file name and its width", tooLarge)
	}
	assertNothingUploaded(t, fixture, result.Uploads)
}

func TestUploadFilesRejectsAnImageTheProcessorCannotRead(t *testing.T) {
	fixture := newUploadFilesFixture()
	decodeFailure := errors.New("corrupt jpeg")
	fixture.imageProcessor.InspectFailure = decodeFailure

	result, err := fixture.upload(uploadParams(models.PurposeProfilePicture, jpegFile("broken.jpg")))

	if _, ok := errors.AsType[e.ErrImageUnreadable](err); !ok {
		t.Fatalf("err = %v, want ErrImageUnreadable", err)
	}
	if !errors.Is(err, decodeFailure) {
		t.Errorf("err = %v, want the decode failure reachable", err)
	}
	assertNothingUploaded(t, fixture, result.Uploads)
}

func TestUploadFilesDiscardsTheOriginalWhenThePreviewFails(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(f uploadFilesFixture)
		wantErr func(err error) bool
	}{
		{
			name:    "preview cannot be created",
			arrange: func(f uploadFilesFixture) { f.imageProcessor.PreviewFailure = errors.New("vips failed") },
			wantErr: errorOfType[e.ErrCreatePreviewFailed](),
		},
		{
			name: "preview cannot be stored",
			arrange: func(f uploadFilesFixture) {
				f.fileStorage.StoreFailure = errStorageDown
				f.fileStorage.StoresBeforeFailure = 1
			},
			wantErr: func(err error) bool {
				_, ok := errors.AsType[e.ErrStoreFileFailed](err)
				return ok && errors.Is(err, errStorageDown)
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newUploadFilesFixture()
			testCase.arrange(fixture)

			result, err := fixture.upload(uploadParams(models.PurposeProfilePicture, jpegFile("me.jpg")))

			if !testCase.wantErr(err) {
				t.Fatalf("err = %v (%T), want the preview failure", err, err)
			}
			if len(fixture.fileStorage.Deleted) != 1 {
				t.Errorf("discarded = %v, want the stored original", fixture.fileStorage.Deleted)
			}
			assertNothingUploaded(t, fixture, result.Uploads)
		})
	}
}

func TestUploadFilesReportsAnOriginalTheStorageRefuses(t *testing.T) {
	fixture := newUploadFilesFixture()
	fixture.fileStorage.StoreFailure = errStorageDown

	result, err := fixture.upload(uploadParams(models.PurposeDocumentAttachment, pdfFile("document.pdf")))

	if _, ok := errors.AsType[e.ErrStoreFileFailed](err); !ok || !errors.Is(err, errStorageDown) {
		t.Fatalf("err = %v, want ErrStoreFileFailed wrapping the storage failure", err)
	}
	if len(fixture.fileStorage.Deleted) != 0 {
		t.Errorf("discarded = %v, want nothing, as nothing was stored", fixture.fileStorage.Deleted)
	}
	assertNothingUploaded(t, fixture, result.Uploads)
}

func TestUploadFilesDiscardsStoredFilesWhenTheRowCannotBeSaved(t *testing.T) {
	fixture := newUploadFilesFixture()
	fixture.repository.SaveFailure = errDatabaseDown

	result, err := fixture.upload(uploadParams(models.PurposeProfilePicture, jpegFile("me.jpg")))

	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want the save failure", err)
	}
	if len(fixture.fileStorage.Deleted) != 2 {
		t.Errorf("discarded = %v, want the original and the preview", fixture.fileStorage.Deleted)
	}
	assertNothingUploaded(t, fixture, result.Uploads)
}

func TestUploadFilesKeepsEarlierUploadsPendingWhenALaterOneFails(t *testing.T) {
	fixture := newUploadFilesFixture()
	fixture.fileStorage.StoreFailure = errStorageDown
	fixture.fileStorage.StoresBeforeFailure = 1

	result, err := fixture.upload(uploadParams(models.PurposeDocumentAttachment,
		pdfFile("first.pdf"), pdfFile("second.pdf")))

	if !errors.Is(err, errStorageDown) {
		t.Fatalf("err = %v, want the storage failure", err)
	}
	if result.Uploads != nil {
		t.Errorf("uploads = %+v, want nil", result.Uploads)
	}
	if len(fixture.repository.Uploads) != 1 || fixture.repository.Uploads[0].IsAttached() {
		t.Errorf("saved uploads = %+v, want the first one left pending for cleanup", fixture.repository.Uploads)
	}
}
