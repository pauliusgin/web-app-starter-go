package mappers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

func TestToAuthChallengeMapsUsedAt(t *testing.T) {
	usedAt := time.Date(2026, time.September, 9, 13, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		storedValue sql.NullTime
		wantUsedAt  time.Time
	}{
		{name: "null used_at means unused", storedValue: sql.NullTime{}},
		{
			name:        "non-null used_at carries the instant",
			storedValue: sql.NullTime{Time: usedAt, Valid: true},
			wantUsedAt:  usedAt,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := AuthChallengeMapper(authChallengeRow(testCase.storedValue))

			if !mapped.UsedAt.Equal(testCase.wantUsedAt) {
				t.Errorf("UsedAt = %v, want %v", mapped.UsedAt, testCase.wantUsedAt)
			}
			if mapped.IsUsed() != testCase.storedValue.Valid {
				t.Errorf("IsUsed() = %t, want %t", mapped.IsUsed(), testCase.storedValue.Valid)
			}
		})
	}
}

func TestToAuthChallengeMapsNullTextColumnsToTheEmptyString(t *testing.T) {
	cases := []struct {
		name       string
		clear      func(row *sqlc.AuthChallenge)
		readMapped func(mapped challenge.Challenge) string
		wantSet    string
	}{
		{
			name:       "token_hash",
			clear:      func(row *sqlc.AuthChallenge) { row.TokenHash = sql.NullString{} },
			readMapped: func(mapped challenge.Challenge) string { return mapped.TokenHash },
			wantSet:    "f2c1a0b9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1",
		},
		{
			name:       "code",
			clear:      func(row *sqlc.AuthChallenge) { row.Code = sql.NullString{} },
			readMapped: func(mapped challenge.Challenge) string { return mapped.Code },
			wantSet:    "418302",
		},
		{
			name:       "provider",
			clear:      func(row *sqlc.AuthChallenge) { row.Provider = sql.NullString{} },
			readMapped: func(mapped challenge.Challenge) string { return mapped.Provider },
			wantSet:    "postmark",
		},
		{
			name:       "reference_id",
			clear:      func(row *sqlc.AuthChallenge) { row.ReferenceID = sql.NullString{} },
			readMapped: func(mapped challenge.Challenge) string { return mapped.ReferenceID },
			wantSet:    "0d4a1f6c-6b53-4a1c-9a55-3f0c2b8e7d41",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			row := authChallengeRow(sql.NullTime{})

			if got := testCase.readMapped(AuthChallengeMapper(row)); got != testCase.wantSet {
				t.Errorf("non-null %s = %q, want %q", testCase.name, got, testCase.wantSet)
			}

			testCase.clear(&row)

			if got := testCase.readMapped(AuthChallengeMapper(row)); got != "" {
				t.Errorf("null %s = %q, want the empty string", testCase.name, got)
			}
		})
	}
}

func TestToAuthChallengeMapsARowWithEveryNullableColumnNull(t *testing.T) {
	row := authChallengeRow(sql.NullTime{})
	row.TokenHash = sql.NullString{}
	row.Code = sql.NullString{}
	row.Provider = sql.NullString{}
	row.ReferenceID = sql.NullString{}

	mapped := AuthChallengeMapper(row)

	if mapped.TokenHash != "" {
		t.Errorf("TokenHash = %q, want the empty string", mapped.TokenHash)
	}
	if mapped.Code != "" {
		t.Errorf("Code = %q, want the empty string", mapped.Code)
	}
	if mapped.Provider != "" {
		t.Errorf("Provider = %q, want the empty string", mapped.Provider)
	}
	if mapped.ReferenceID != "" {
		t.Errorf("ReferenceID = %q, want the empty string", mapped.ReferenceID)
	}
	if !mapped.UsedAt.IsZero() {
		t.Errorf("UsedAt = %v, want the zero time", mapped.UsedAt)
	}
	if mapped.IsUsed() {
		t.Error("IsUsed() = true, want false for a row with a null used_at")
	}
}

func TestToAuthChallengeMapsEveryColumn(t *testing.T) {
	usedAt := time.Date(2026, time.September, 9, 13, 0, 0, 0, time.UTC)
	row := authChallengeRow(sql.NullTime{Time: usedAt, Valid: true})

	mapped := AuthChallengeMapper(row)

	if mapped.ID != row.ID {
		t.Errorf("ID = %s, want %s", mapped.ID, row.ID)
	}
	if mapped.UserID != row.UserID {
		t.Errorf("UserID = %s, want %s", mapped.UserID, row.UserID)
	}
	if mapped.TargetType != challenge.TargetType(row.TargetType) {
		t.Errorf("TargetType = %q, want %q", mapped.TargetType, row.TargetType)
	}
	if mapped.Target != row.Target {
		t.Errorf("Target = %q, want %q", mapped.Target, row.Target)
	}
	if mapped.Purpose != challenge.Purpose(row.Purpose) {
		t.Errorf("Purpose = %q, want %q", mapped.Purpose, row.Purpose)
	}
	if mapped.TokenHash != row.TokenHash.String {
		t.Errorf("TokenHash = %q, want %q", mapped.TokenHash, row.TokenHash.String)
	}
	if mapped.Code != row.Code.String {
		t.Errorf("Code = %q, want %q", mapped.Code, row.Code.String)
	}
	if mapped.Provider != row.Provider.String {
		t.Errorf("Provider = %q, want %q", mapped.Provider, row.Provider.String)
	}
	if mapped.ReferenceID != row.ReferenceID.String {
		t.Errorf("ReferenceID = %q, want %q", mapped.ReferenceID, row.ReferenceID.String)
	}
	if !mapped.CreatedAt.Equal(row.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", mapped.CreatedAt, row.CreatedAt)
	}
	if !mapped.ExpiresAt.Equal(row.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", mapped.ExpiresAt, row.ExpiresAt)
	}
	if !mapped.UsedAt.Equal(usedAt) {
		t.Errorf("UsedAt = %v, want %v", mapped.UsedAt, usedAt)
	}
}

func TestToAuthChallengeMapsTargetTypeAndPurposeIntoTheNamedDomainTypes(t *testing.T) {
	row := authChallengeRow(sql.NullTime{})

	mapped := AuthChallengeMapper(row)

	if mapped.TargetType != challenge.TargetTypeEmail {
		t.Errorf("TargetType = %q, want %q", mapped.TargetType, challenge.TargetTypeEmail)
	}
	if mapped.Purpose != challenge.PurposeEmailVerification {
		t.Errorf("Purpose = %q, want %q", mapped.Purpose, challenge.PurposeEmailVerification)
	}
}

func authChallengeRow(usedAt sql.NullTime) sqlc.AuthChallenge {
	createdAt := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)

	return sqlc.AuthChallenge{
		ID:         uuid.MustParse("3f8b1c94-9a2d-4f61-8c07-5b1e2d7a6c30"),
		UserID:     uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15"),
		TargetType: string(challenge.TargetTypeEmail),
		Target:     "someone@example.com",
		Purpose:    string(challenge.PurposeEmailVerification),
		TokenHash: sql.NullString{
			String: "f2c1a0b9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1",
			Valid:  true,
		},
		Code:        sql.NullString{String: "418302", Valid: true},
		Provider:    sql.NullString{String: "postmark", Valid: true},
		ReferenceID: sql.NullString{String: "0d4a1f6c-6b53-4a1c-9a55-3f0c2b8e7d41", Valid: true},
		CreatedAt:   createdAt,
		ExpiresAt:   createdAt.Add(challenge.Lifetime),
		UsedAt:      usedAt,
	}
}
