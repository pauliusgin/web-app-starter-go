// Package firebase stores the domain's uploaded files in a Firebase Storage bucket.
package firebase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/option"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	filestorage "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
)

const downloadBaseURL = "https://firebasestorage.googleapis.com/v0"

// downloadTokenMetadataKey is the object metadata Firebase checks a download URL's token against.
const downloadTokenMetadataKey = "firebaseStorageDownloadTokens"

var _ filestorage.FileStorage = StorageGateway{}

// StorageGateway stores and deletes files in one Firebase Storage bucket.
type StorageGateway struct {
	client     *storage.Client
	bucketName string
}

// StorageGatewayParams is the base64 service account JSON and the bucket the gateway works in.
type StorageGatewayParams struct {
	EncodedCredentials string
	BucketName         string
}

// NewStorageGateway decodes the service account and builds the one client the process reuses.
func NewStorageGateway(ctx context.Context, params StorageGatewayParams) (StorageGateway, error) {
	if params.BucketName == "" {
		return StorageGateway{}, ErrBucketNameMissing{}
	}

	credentials, err := base64.StdEncoding.DecodeString(params.EncodedCredentials)
	if err != nil {
		return StorageGateway{}, ErrCredentialsUndecodable{Cause: err}
	}

	client, err := storage.NewClient(ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, credentials))
	if err != nil {
		return StorageGateway{}, ErrClientUnavailable{Cause: err}
	}

	return StorageGateway{client: client, bucketName: params.BucketName}, nil
}

// Close releases the client's connections.
func (g StorageGateway) Close() error {
	return g.client.Close()
}

// StoreFile writes the file with a fresh download token and answers with its tokenised download URL.
func (g StorageGateway) StoreFile(
	ctx context.Context,
	params filestorage.StoreFileParams,
) (filestorage.StoredFile, error) {
	downloadToken := uuid.NewString()

	writer := g.client.Bucket(g.bucketName).Object(params.Path).NewWriter(ctx)
	writer.ContentType = params.ContentType
	writer.Metadata = map[string]string{downloadTokenMetadataKey: downloadToken}

	if _, err := writer.Write(params.Content); err != nil {
		_ = writer.Close()
		return filestorage.StoredFile{}, ErrObjectWriteFailed{Path: params.Path, Cause: err}
	}
	// The object exists only once Close succeeds; Write alone may have buffered everything.
	if err := writer.Close(); err != nil {
		return filestorage.StoredFile{}, ErrObjectWriteFailed{Path: params.Path, Cause: err}
	}

	downloadURL := BuildDownloadURL(BuildDownloadURLParams{
		BucketName:    g.bucketName,
		Path:          params.Path,
		DownloadToken: downloadToken,
	})

	return filestorage.StoredFile{URL: downloadURL}, nil
}

// DeleteFile removes the object at path, reporting ErrStoredFileNotFound when none exists.
func (g StorageGateway) DeleteFile(ctx context.Context, path string) error {
	err := g.client.Bucket(g.bucketName).Object(path).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return e.ErrStoredFileNotFound{Path: path}
	}
	if err != nil {
		return ErrObjectDeleteFailed{Path: path, Cause: err}
	}

	return nil
}

// BuildDownloadURLParams names the object and the token its download URL carries.
type BuildDownloadURLParams struct {
	BucketName    string
	Path          string
	DownloadToken string
}

// BuildDownloadURL returns the public Firebase download URL, the object path escaped as one segment.
func BuildDownloadURL(params BuildDownloadURLParams) string {
	query := url.Values{}
	query.Set("alt", "media")
	query.Set("token", params.DownloadToken)

	return fmt.Sprintf("%s/b/%s/o/%s?%s",
		downloadBaseURL, url.PathEscape(params.BucketName), url.PathEscape(params.Path), query.Encode())
}
