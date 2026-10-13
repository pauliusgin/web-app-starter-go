package refreshtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

func TestDeleteUnusedRefreshTokensDeletesOnlyTokensPastTheThreshold(t *testing.T) {
	cases := []struct {
		name        string
		params      seedTokenParams
		wantDeleted bool
	}{
		{
			name:   "live",
			params: seedTokenParams{UserID: uuid.New()},
		},
		{
			name: "expired an hour ago",
			params: seedTokenParams{
				UserID:    uuid.New(),
				ExpiresAt: fixedRotatedAt.Add(-time.Hour),
			},
			wantDeleted: true,
		},
		{
			name: "revoked a day ago, inside the grace window",
			params: seedTokenParams{
				UserID:    uuid.New(),
				RevokedAt: fixedRotatedAt.Add(-24 * time.Hour),
			},
		},
		{
			name: "revoked thirty days ago, past the grace window",
			params: seedTokenParams{
				UserID:    uuid.New(),
				RevokedAt: fixedRotatedAt.Add(-30 * 24 * time.Hour),
			},
			wantDeleted: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newInMemoryTokenStore()
			plaintext := store.seedToken(t, testCase.params)

			deletedCount, err := DeleteUnusedRefreshTokens(context.Background(),
				UnusedDeletionDependencies{WriteRepository: store},
				UnusedDeletionParams{CleanUpThreshold: cleanUpThresholdInDays})
			if err != nil {
				t.Fatalf("DeleteUnusedRefreshTokens: %v", err)
			}

			_, stillStored := store.rows[token.Hash(plaintext)]
			if stillStored == testCase.wantDeleted {
				t.Errorf("row still stored = %t, want deleted = %t", stillStored, testCase.wantDeleted)
			}
			if testCase.wantDeleted && deletedCount != 1 {
				t.Errorf("deleted count = %d, want 1", deletedCount)
			}
			if !testCase.wantDeleted && deletedCount != 0 {
				t.Errorf("deleted count = %d, want 0", deletedCount)
			}
		})
	}
}

func TestDeleteUnusedRefreshTokensCountsEveryRowItRemoved(t *testing.T) {
	store := newInMemoryTokenStore()
	userID := uuid.New()

	store.seedToken(t, seedTokenParams{UserID: userID, ExpiresAt: fixedRotatedAt.Add(-time.Hour)})
	store.seedToken(t, seedTokenParams{
		UserID:    userID,
		RevokedAt: fixedRotatedAt.Add(-30 * 24 * time.Hour),
	})
	store.seedToken(t, seedTokenParams{UserID: userID})

	deletedCount, err := DeleteUnusedRefreshTokens(context.Background(),
		UnusedDeletionDependencies{WriteRepository: store},
		UnusedDeletionParams{CleanUpThreshold: cleanUpThresholdInDays})
	if err != nil {
		t.Fatalf("DeleteUnusedRefreshTokens: %v", err)
	}

	if deletedCount != 2 {
		t.Errorf("deleted count = %d, want 2", deletedCount)
	}
	if len(store.tokensForUser(userID)) != 1 {
		t.Errorf("rows left = %d, want the one live row", len(store.tokensForUser(userID)))
	}
}

func TestDeleteUnusedRefreshTokensSurfacesADeleteFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	store.deleteUnusedFailure = ErrDeleteUnusedRefreshTokensFailed{Cause: cause}

	deletedCount, err := DeleteUnusedRefreshTokens(context.Background(),
		UnusedDeletionDependencies{WriteRepository: store},
		UnusedDeletionParams{CleanUpThreshold: cleanUpThresholdInDays})

	deleteFailed := ErrDeleteUnusedRefreshTokensFailed{}
	if !errors.As(err, &deleteFailed) {
		t.Fatalf("error = %v, want ErrDeleteUnusedRefreshTokensFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the delete failure does not unwrap to the driver error")
	}
	if deletedCount != 0 {
		t.Errorf("deleted count = %d, want 0 on failure", deletedCount)
	}
}
