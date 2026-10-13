package mappers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
)

var fixedRevokedAt = time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)

func TestToRefreshTokenMapsEveryColumn(t *testing.T) {
	row := refreshTokenRow(sql.NullTime{Time: fixedRevokedAt, Valid: true})

	mapped := RefreshTokenMapper(row)

	if mapped.ID != row.ID {
		t.Errorf("ID = %s, want %s", mapped.ID, row.ID)
	}
	if mapped.UserID != row.UserID {
		t.Errorf("UserID = %s, want %s", mapped.UserID, row.UserID)
	}
	if mapped.TokenHash != row.TokenHash {
		t.Errorf("TokenHash = %q, want %q", mapped.TokenHash, row.TokenHash)
	}
	if !mapped.CreatedAt.Equal(row.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", mapped.CreatedAt, row.CreatedAt)
	}
	if !mapped.ExpiresAt.Equal(row.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", mapped.ExpiresAt, row.ExpiresAt)
	}
	if !mapped.RevokedAt.Equal(fixedRevokedAt) {
		t.Errorf("RevokedAt = %v, want %v", mapped.RevokedAt, fixedRevokedAt)
	}
}

func TestToRefreshTokenMapsNullRevokedAtToLive(t *testing.T) {
	mapped := RefreshTokenMapper(refreshTokenRow(sql.NullTime{}))

	if mapped.IsRevoked() {
		t.Error("IsRevoked = true, want false for a null revoked_at")
	}
}

func refreshTokenRow(revokedAt sql.NullTime) sqlc.RefreshToken {
	createdAt := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

	return sqlc.RefreshToken{
		ID:        uuid.MustParse("7b2f6f7a-2a1e-4c3d-9f10-6d4c8b1e5a20"),
		UserID:    uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15"),
		TokenHash: "f2c1a0b9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1",
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(30 * 24 * time.Hour),
		RevokedAt: revokedAt,
	}
}
