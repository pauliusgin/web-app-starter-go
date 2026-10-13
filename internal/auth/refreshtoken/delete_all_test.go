package refreshtoken

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestDeleteUserRefreshTokensSurfacesADeleteFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	store.deleteFailure = ErrDeleteRefreshTokensFailed{Cause: cause}

	err := DeleteUserRefreshTokens(context.Background(),
		DeleteUserRefreshTokensDependencies{WriteRepository: store},
		DeleteUserRefreshTokensParams{UserID: uuid.New()})

	deleteFailed := ErrDeleteRefreshTokensFailed{}
	if !errors.As(err, &deleteFailed) {
		t.Fatalf("error = %v, want ErrDeleteRefreshTokensFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the delete failure does not unwrap to the driver error")
	}
}
