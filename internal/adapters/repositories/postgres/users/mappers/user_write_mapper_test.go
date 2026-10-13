package mappers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
)

var mappedUserID = uuid.MustParse("8a2f4c61-3d90-4e17-b5c2-6f1a9d7e0b34")

var mappedCreatedAt = time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

var mappedUpdatedAt = time.Date(2026, time.September, 1, 12, 0, 1, 0, time.UTC)

var mappedVerifiedAt = time.Date(2026, time.September, 1, 12, 3, 0, 0, time.UTC)

var mappedSignedInAt = time.Date(2026, time.September, 2, 8, 30, 0, 0, time.UTC)

const mappedEmail = "accountant@example.com"

func TestUserFromIDMapperMapsEmailVerifiedAt(t *testing.T) {
	cases := []struct {
		name         string
		storedValue  sql.NullTime
		wantStamp    time.Time
		wantVerified bool
	}{
		{name: "null email_verified_at means unverified"},
		{
			name:         "non-null email_verified_at carries the instant",
			storedValue:  sql.NullTime{Time: mappedVerifiedAt, Valid: true},
			wantStamp:    mappedVerifiedAt,
			wantVerified: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := UserFromIDMapper(sqlc.FindUserByIDRow{
				ID:              mappedUserID,
				Email:           mappedEmail,
				CreatedAt:       mappedCreatedAt,
				UpdatedAt:       mappedUpdatedAt,
				EmailVerifiedAt: testCase.storedValue,
			})

			if !mapped.EmailVerifiedAt.Equal(testCase.wantStamp) {
				t.Errorf("EmailVerifiedAt = %v, want %v", mapped.EmailVerifiedAt, testCase.wantStamp)
			}
			if mapped.IsEmailVerified() != testCase.wantVerified {
				t.Errorf("IsEmailVerified() = %t, want %t",
					mapped.IsEmailVerified(), testCase.wantVerified)
			}
		})
	}
}

func TestUserFromEmailMapperMapsEveryColumnTheDomainKeeps(t *testing.T) {
	mapped := UserFromEmailMapper(sqlc.FindUserByEmailRow{
		ID:              mappedUserID,
		Email:           mappedEmail,
		CreatedAt:       mappedCreatedAt,
		UpdatedAt:       mappedUpdatedAt,
		EmailVerifiedAt: sql.NullTime{Time: mappedVerifiedAt, Valid: true},
	})

	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
	if mapped.Email != mappedEmail {
		t.Errorf("Email = %q, want %q", mapped.Email, mappedEmail)
	}
	if !mapped.CreatedAt.Equal(mappedCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", mapped.CreatedAt, mappedCreatedAt)
	}
	if !mapped.UpdatedAt.Equal(mappedUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", mapped.UpdatedAt, mappedUpdatedAt)
	}
	if !mapped.EmailVerifiedAt.Equal(mappedVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want %v", mapped.EmailVerifiedAt, mappedVerifiedAt)
	}
}

func TestUserFromEmailMapperMapsANullEmailVerifiedAtToUnverified(t *testing.T) {
	mapped := UserFromEmailMapper(sqlc.FindUserByEmailRow{
		ID:        mappedUserID,
		Email:     mappedEmail,
		CreatedAt: mappedCreatedAt,
		UpdatedAt: mappedUpdatedAt,
	})

	if mapped.IsEmailVerified() {
		t.Error("IsEmailVerified() = true, want false for a null email_verified_at")
	}
}

func TestCreateUserMapperLeavesANewUserUnverified(t *testing.T) {
	mapped := CreateUserMapper(sqlc.CreateUserRow{
		ID:        mappedUserID,
		Email:     mappedEmail,
		CreatedAt: mappedCreatedAt,
		UpdatedAt: mappedUpdatedAt,
	})

	if mapped.IsEmailVerified() {
		t.Error("IsEmailVerified() = true, want a freshly created user to be unverified")
	}
	if !mapped.EmailVerifiedAt.IsZero() {
		t.Errorf("EmailVerifiedAt = %v, want the zero time", mapped.EmailVerifiedAt)
	}
}

func TestMarkUserEmailAsVerifiedMapperMapsEveryColumnTheDomainKeeps(t *testing.T) {
	mapped := MarkUserEmailAsVerifiedMapper(verifiedUserRow())

	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
	if mapped.Email != mappedEmail {
		t.Errorf("Email = %q, want %q", mapped.Email, mappedEmail)
	}
	if !mapped.CreatedAt.Equal(mappedCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", mapped.CreatedAt, mappedCreatedAt)
	}
	if !mapped.UpdatedAt.Equal(mappedUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", mapped.UpdatedAt, mappedUpdatedAt)
	}
	if !mapped.EmailVerifiedAt.Equal(mappedVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want %v", mapped.EmailVerifiedAt, mappedVerifiedAt)
	}
	if !mapped.IsEmailVerified() {
		t.Error("IsEmailVerified() = false, want true")
	}
}

func TestMarkUserEmailAsVerifiedMapperMapsANullEmailVerifiedAtToTheZeroTime(t *testing.T) {
	row := verifiedUserRow()
	row.EmailVerifiedAt = sql.NullTime{}

	mapped := MarkUserEmailAsVerifiedMapper(row)

	if !mapped.EmailVerifiedAt.IsZero() {
		t.Errorf("EmailVerifiedAt = %v, want the zero time", mapped.EmailVerifiedAt)
	}
	if mapped.IsEmailVerified() {
		t.Error("IsEmailVerified() = true, want false for a null email_verified_at")
	}
}

func verifiedUserRow() sqlc.MarkUserEmailAsVerifiedRow {
	return sqlc.MarkUserEmailAsVerifiedRow{
		ID:              mappedUserID,
		Email:           mappedEmail,
		CreatedAt:       mappedCreatedAt,
		UpdatedAt:       mappedUpdatedAt,
		EmailVerifiedAt: sql.NullTime{Time: mappedVerifiedAt, Valid: true},
	}
}

func TestUserCredentialsFromEmailMapperKeepsThePasswordHash(t *testing.T) {
	const storedHash = "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$aGFzaA"

	mapped := UserCredentialsFromEmailMapper(sqlc.FindUserCredentialsByEmailRow{
		ID:           mappedUserID,
		Email:        mappedEmail,
		PasswordHash: storedHash,
	})

	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
	if mapped.Email != mappedEmail {
		t.Errorf("Email = %q, want %q", mapped.Email, mappedEmail)
	}
	if mapped.PasswordHash != storedHash {
		t.Errorf("PasswordHash = %q, want %q", mapped.PasswordHash, storedHash)
	}
}

func TestUserCredentialsFromIDMapperKeepsThePasswordHash(t *testing.T) {
	const storedHash = "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$aGFzaA"

	mapped := UserCredentialsFromIDMapper(sqlc.FindUserCredentialsByIDRow{
		ID:           mappedUserID,
		Email:        mappedEmail,
		PasswordHash: storedHash,
	})

	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
	if mapped.Email != mappedEmail {
		t.Errorf("Email = %q, want %q", mapped.Email, mappedEmail)
	}
	if mapped.PasswordHash != storedHash {
		t.Errorf("PasswordHash = %q, want %q", mapped.PasswordHash, storedHash)
	}
}

func TestMarkUserSignedInMapperMapsEveryColumnTheDomainKeeps(t *testing.T) {
	mapped := MarkUserSignedInMapper(sqlc.MarkUserSignedInRow{
		ID:              mappedUserID,
		Email:           mappedEmail,
		CreatedAt:       mappedCreatedAt,
		UpdatedAt:       mappedUpdatedAt,
		EmailVerifiedAt: sql.NullTime{Time: mappedVerifiedAt, Valid: true},
		SigninAt:        sql.NullTime{Time: mappedSignedInAt, Valid: true},
	})

	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
	if mapped.Email != mappedEmail {
		t.Errorf("Email = %q, want %q", mapped.Email, mappedEmail)
	}
	if !mapped.CreatedAt.Equal(mappedCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", mapped.CreatedAt, mappedCreatedAt)
	}
	if !mapped.UpdatedAt.Equal(mappedUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", mapped.UpdatedAt, mappedUpdatedAt)
	}
	if !mapped.EmailVerifiedAt.Equal(mappedVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want %v", mapped.EmailVerifiedAt, mappedVerifiedAt)
	}
	if !mapped.SigninAt.Equal(mappedSignedInAt) {
		t.Errorf("SigninAt = %v, want %v", mapped.SigninAt, mappedSignedInAt)
	}
}

func TestUserFromEmailMapperMapsSigninAt(t *testing.T) {
	cases := []struct {
		name        string
		storedValue sql.NullTime
		wantStamp   time.Time
	}{
		{name: "null signin_at means never signed in"},
		{
			name:        "non-null signin_at carries the instant",
			storedValue: sql.NullTime{Time: mappedSignedInAt, Valid: true},
			wantStamp:   mappedSignedInAt,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := UserFromEmailMapper(sqlc.FindUserByEmailRow{
				ID:       mappedUserID,
				Email:    mappedEmail,
				SigninAt: testCase.storedValue,
			})

			if !mapped.SigninAt.Equal(testCase.wantStamp) {
				t.Errorf("SigninAt = %v, want %v", mapped.SigninAt, testCase.wantStamp)
			}
		})
	}
}

func TestChangeUserEmailMapperMapsEveryColumnTheDomainKeeps(t *testing.T) {
	const changedEmail = "bookkeeper@example.com"

	mapped := ChangeUserEmailMapper(sqlc.ChangeUserEmailRow{
		ID:              mappedUserID,
		Email:           changedEmail,
		CreatedAt:       mappedCreatedAt,
		UpdatedAt:       mappedUpdatedAt,
		EmailVerifiedAt: sql.NullTime{Time: mappedVerifiedAt, Valid: true},
		SigninAt:        sql.NullTime{Time: mappedSignedInAt, Valid: true},
	})

	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
	if mapped.Email != changedEmail {
		t.Errorf("Email = %q, want %q", mapped.Email, changedEmail)
	}
	if !mapped.CreatedAt.Equal(mappedCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", mapped.CreatedAt, mappedCreatedAt)
	}
	if !mapped.UpdatedAt.Equal(mappedUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", mapped.UpdatedAt, mappedUpdatedAt)
	}
	if !mapped.EmailVerifiedAt.Equal(mappedVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want %v", mapped.EmailVerifiedAt, mappedVerifiedAt)
	}
	if !mapped.SigninAt.Equal(mappedSignedInAt) {
		t.Errorf("SigninAt = %v, want %v", mapped.SigninAt, mappedSignedInAt)
	}
}
