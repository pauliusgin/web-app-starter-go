package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestDetectFileFormatSniffsAcceptedFormatsFromContent(t *testing.T) {
	cases := []struct {
		name         string
		content      []byte
		wantAccepted bool
		wantMimeType string
		wantKind     Kind
		wantExt      string
	}{
		{name: "jpeg", content: []byte("\xFF\xD8\xFF\xE0rest"), wantAccepted: true, wantMimeType: "image/jpeg", wantKind: KindImage, wantExt: "jpg"},
		{name: "png", content: []byte("\x89PNG\r\n\x1a\nrest"), wantAccepted: true, wantMimeType: "image/png", wantKind: KindImage, wantExt: "png"},
		{name: "webp", content: []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), wantAccepted: true, wantMimeType: "image/webp", wantKind: KindImage, wantExt: "webp"},
		{name: "pdf", content: []byte("%PDF-1.7\n"), wantAccepted: true, wantMimeType: "application/pdf", wantKind: KindDocument, wantExt: "pdf"},
		{name: "plain text", content: []byte("just some notes"), wantAccepted: false, wantMimeType: "text/plain; charset=utf-8"},
		{name: "gif", content: []byte("GIF89a rest"), wantAccepted: false, wantMimeType: "image/gif"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			format, detectedMimeType, accepted := DetectFileFormat(testCase.content)

			if detectedMimeType != testCase.wantMimeType {
				t.Errorf("detected = %q, want %q", detectedMimeType, testCase.wantMimeType)
			}
			if accepted != testCase.wantAccepted {
				t.Fatalf("accepted = %v, want %v", accepted, testCase.wantAccepted)
			}
			if !testCase.wantAccepted {
				return
			}
			if format.Kind != testCase.wantKind || format.Extension != testCase.wantExt {
				t.Errorf("format = %+v, want kind %q extension %q", format, testCase.wantKind, testCase.wantExt)
			}
		})
	}
}

func TestRulesForPurposeAcceptsOnlyTheKindsEachPurposeAllows(t *testing.T) {
	cases := []struct {
		name        string
		purpose     Purpose
		kind        Kind
		wantAllowed bool
	}{
		{name: "profile picture image", purpose: PurposeProfilePicture, kind: KindImage, wantAllowed: true},
		{name: "profile picture document", purpose: PurposeProfilePicture, kind: KindDocument, wantAllowed: false},
		{name: "document attachment image", purpose: PurposeDocumentAttachment, kind: KindImage, wantAllowed: true},
		{name: "document attachment document", purpose: PurposeDocumentAttachment, kind: KindDocument, wantAllowed: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rules, known := RulesForPurpose(testCase.purpose)
			if !known {
				t.Fatalf("RulesForPurpose(%q) reported an unknown purpose", testCase.purpose)
			}

			if allowed := rules.AllowsKind(testCase.kind); allowed != testCase.wantAllowed {
				t.Errorf("AllowsKind(%q) = %v, want %v", testCase.kind, allowed, testCase.wantAllowed)
			}
		})
	}
}

func TestRulesForPurposeRejectsAnUnknownPurpose(t *testing.T) {
	if _, known := RulesForPurpose("cover_photo"); known {
		t.Error("RulesForPurpose(\"cover_photo\") reported a known purpose")
	}
}

func TestBuildStoragePathsPlacesFilesUnderEnvironmentOwnerPurposeAndID(t *testing.T) {
	paths := BuildStoragePaths(StoragePathsParams{
		AppEnv:    "development",
		UserID:    uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15"),
		Purpose:   PurposeProfilePicture,
		UploadID:  uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f"),
		Extension: "png",
	})

	directory := "uploads/development/users/1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15/profile_picture/" +
		"6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f"
	if paths.Original != directory+"/original.png" {
		t.Errorf("Original = %q, want %q", paths.Original, directory+"/original.png")
	}
	if paths.Preview != directory+"/preview.webp" {
		t.Errorf("Preview = %q, want %q", paths.Preview, directory+"/preview.webp")
	}
}

func TestStoredPathsListsThePreviewOnlyWhenOneExists(t *testing.T) {
	document := Upload{OriginalPath: "a/original.pdf"}
	if paths := document.StoredPaths(); len(paths) != 1 || paths[0] != "a/original.pdf" {
		t.Errorf("document StoredPaths = %v, want only the original", paths)
	}

	image := Upload{OriginalPath: "b/original.png", PreviewPath: "b/preview.webp"}
	if paths := image.StoredPaths(); len(paths) != 2 || paths[0] != "b/original.png" || paths[1] != "b/preview.webp" {
		t.Errorf("image StoredPaths = %v, want original then preview", paths)
	}
}
