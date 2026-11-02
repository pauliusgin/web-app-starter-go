// Package uploadrepositorytest holds the behaviour every upload repository pair must share.
package uploadrepositorytest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
)

// ContractSubject is one fresh, empty store seen through both ports, and a way to create the users its uploads
// belong to. Writes through WriteRepository must be visible through ReadRepository.
type ContractSubject struct {
	WriteRepository repositories.UploadWriteRepository
	ReadRepository  read.UploadReadRepository
	SeedUser        func(t *testing.T) uuid.UUID
}

// RunUploadRepositoryContractParams builds a fresh ContractSubject for every contract case.
type RunUploadRepositoryContractParams struct {
	NewSubject func(t *testing.T) ContractSubject
}

// RunUploadRepositoryContract runs the shared cases, so the in-memory fake cannot drift from the SQL.
func RunUploadRepositoryContract(t *testing.T, params RunUploadRepositoryContractParams) {
	cases := []struct {
		name string
		run  func(t *testing.T, harness contractHarness)
	}{
		{name: "saved upload is read back only by its owner", run: savedUploadIsReadBackOnlyByItsOwner},
		{name: "a missing preview reads back as nil", run: aMissingPreviewReadsBackAsNil},
		{name: "a fresh pending upload is not stale yet", run: aFreshPendingUploadIsNotStaleYet},
		{name: "deleting removes the upload and tolerates a missing one", run: deletingRemovesTheUpload},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			subject := params.NewSubject(t)
			testCase.run(t, contractHarness{subject: subject, ctx: context.Background(), userID: subject.SeedUser(t)})
		})
	}
}

type contractHarness struct {
	subject ContractSubject
	ctx     context.Context
	userID  uuid.UUID
}

func (h contractHarness) save(t *testing.T, previewURL string) models.Upload {
	t.Helper()

	return h.saveNewUpload(t, h.newUpload(previewURL))
}

func (h contractHarness) newUpload(previewURL string) models.NewUpload {
	id := uuid.New()
	previewPath := ""
	if previewURL != "" {
		previewPath = "previews/" + id.String() + ".webp"
	}

	return models.NewUpload{
		ID:           id,
		UserID:       h.userID,
		Purpose:      models.PurposeProfilePicture,
		Kind:         models.KindImage,
		MimeType:     "image/png",
		SizeBytes:    2048,
		Width:        64,
		Height:       64,
		OriginalPath: "originals/" + id.String() + ".png",
		OriginalURL:  "https://files.example/originals/" + id.String() + ".png",
		PreviewPath:  previewPath,
		PreviewURL:   previewURL,
	}
}

func (h contractHarness) saveNewUpload(t *testing.T, newUpload models.NewUpload) models.Upload {
	t.Helper()

	saved, err := h.subject.WriteRepository.SaveUpload(h.ctx, newUpload)
	if err != nil {
		t.Fatalf("SaveUpload: %v", err)
	}

	return saved
}

func (h contractHarness) read(userID uuid.UUID, uploadID uuid.UUID) (readmodels.UploadReadModel, error) {
	return h.subject.ReadRepository.GetUploadReadModelByID(h.ctx, readmodels.GetUploadReadModelByIDParams{
		ID:     uploadID,
		UserID: userID,
	})
}

func assertUploadNotFound(t *testing.T, err error) {
	t.Helper()

	if _, notFound := errors.AsType[e.ErrUploadNotFound](err); !notFound {
		t.Errorf("error = %v, want ErrUploadNotFound", err)
	}
}

func savedUploadIsReadBackOnlyByItsOwner(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "https://files.example/preview.webp")

	readBack, err := harness.read(harness.userID, saved.ID)
	if err != nil {
		t.Fatalf("GetUploadReadModelByID: %v", err)
	}
	want := readmodels.UploadReadModel{
		ID:          saved.ID,
		UserID:      harness.userID,
		Purpose:     string(models.PurposeProfilePicture),
		Kind:        string(models.KindImage),
		MimeType:    "image/png",
		OriginalURL: saved.OriginalURL,
	}
	gotPreview := readBack.PreviewURL
	readBack.PreviewURL = nil
	if readBack != want {
		t.Errorf("read back = %+v, want %+v", readBack, want)
	}
	if gotPreview == nil || *gotPreview != "https://files.example/preview.webp" {
		t.Errorf("PreviewURL = %v, want the preview URL", gotPreview)
	}

	_, err = harness.read(uuid.New(), saved.ID)
	assertUploadNotFound(t, err)
}

func aMissingPreviewReadsBackAsNil(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "")

	readBack, err := harness.read(harness.userID, saved.ID)
	if err != nil {
		t.Fatalf("GetUploadReadModelByID: %v", err)
	}
	if readBack.PreviewURL != nil {
		t.Errorf("PreviewURL = %q, want nil for an upload without a preview", *readBack.PreviewURL)
	}
}

func aFreshPendingUploadIsNotStaleYet(t *testing.T, harness contractHarness) {
	harness.save(t, "")

	stale, err := harness.subject.WriteRepository.ListUnusedPendingUploads(harness.ctx)
	if err != nil {
		t.Fatalf("ListUnusedPendingUploads: %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("stale uploads = %+v, want none younger than models.PendingUploadRetention", stale)
	}
}

func deletingRemovesTheUpload(t *testing.T, harness contractHarness) {
	saved := harness.save(t, "")

	if err := harness.subject.WriteRepository.DeleteUploadByID(harness.ctx, saved.ID); err != nil {
		t.Fatalf("DeleteUploadByID: %v", err)
	}
	_, err := harness.read(harness.userID, saved.ID)
	assertUploadNotFound(t, err)

	if err := harness.subject.WriteRepository.DeleteUploadByID(harness.ctx, saved.ID); err != nil {
		t.Errorf("DeleteUploadByID(already gone) = %v, want nil", err)
	}
}
