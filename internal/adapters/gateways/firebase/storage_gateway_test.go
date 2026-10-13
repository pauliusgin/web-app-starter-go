package firebase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	filestorage "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
)

func TestBuildDownloadURLEscapesThePathAsOneSegment(t *testing.T) {
	downloadURL := BuildDownloadURL(BuildDownloadURLParams{
		BucketName:    "web_app_starter.firebasestorage.app",
		Path:          "uploads/test/users/42/profile_picture/7/original file.jpg",
		DownloadToken: "token-1",
	})

	want := "https://firebasestorage.googleapis.com/v0/b/web_app_starter.firebasestorage.app/o/" +
		"uploads%2Ftest%2Fusers%2F42%2Fprofile_picture%2F7%2Foriginal%20file.jpg?alt=media&token=token-1"
	if downloadURL != want {
		t.Errorf("BuildDownloadURL =\n%s\nwant\n%s", downloadURL, want)
	}
}

func TestNewStorageGatewayRejectsMissingConfiguration(t *testing.T) {
	cases := []struct {
		name    string
		params  StorageGatewayParams
		wantErr func(err error) bool
	}{
		{
			name:   "no bucket",
			params: StorageGatewayParams{EncodedCredentials: "e30="},
			wantErr: func(err error) bool {
				_, ok := errors.AsType[ErrBucketNameMissing](err)
				return ok
			},
		},
		{
			name:   "credentials not base64",
			params: StorageGatewayParams{EncodedCredentials: "{not base64}", BucketName: "bucket"},
			wantErr: func(err error) bool {
				_, ok := errors.AsType[ErrCredentialsUndecodable](err)
				return ok
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewStorageGateway(context.Background(), testCase.params)

			if !testCase.wantErr(err) {
				t.Errorf("err = %v (%T), want the rejection this case names", err, err)
			}
		})
	}
}

// Live round trip against the real bucket; runs only when both variables are exported.
func TestStorageGatewayStoresServesAndDeletesAFile(t *testing.T) {
	encodedCredentials := os.Getenv("GOOGLE_FIREBASE_CREDENTIALS")
	bucketName := os.Getenv("GOOGLE_FIREBASE_STORAGE_BUCKET")
	if encodedCredentials == "" || bucketName == "" {
		t.Skip("GOOGLE_FIREBASE_CREDENTIALS and GOOGLE_FIREBASE_STORAGE_BUCKET are not exported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gateway, err := NewStorageGateway(ctx, StorageGatewayParams{
		EncodedCredentials: encodedCredentials,
		BucketName:         bucketName,
	})
	if err != nil {
		t.Fatalf("NewStorageGateway: %v", err)
	}
	defer gateway.Close()

	path := "uploads/gateway-test/" + uuid.NewString() + "/original.txt"
	content := []byte("firebase gateway round trip")

	stored, err := gateway.StoreFile(ctx, filestorage.StoreFileParams{
		Path:        path,
		ContentType: "text/plain",
		Content:     content,
	})
	if err != nil {
		t.Fatalf("StoreFile: %v", err)
	}

	downloaded := downloadFile(t, ctx, stored.URL)
	if !bytes.Equal(downloaded, content) {
		t.Errorf("downloaded = %q, want %q", downloaded, content)
	}

	if err := gateway.DeleteFile(ctx, path); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, ok := errors.AsType[e.ErrStoredFileNotFound](gateway.DeleteFile(ctx, path)); !ok {
		t.Error("a second delete did not report ErrStoredFileNotFound")
	}
}

func downloadFile(t *testing.T, ctx context.Context, downloadURL string) []byte {
	t.Helper()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		t.Fatalf("building the download request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("downloading %s: %v", downloadURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d, want 200", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading the download: %v", err)
	}

	return body
}
