// Package refreshtokenrepositorytest holds the behaviour every refresh token repository pair must share.
package refreshtokenrepositorytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

// ContractSubject is one fresh store seen through both ports, and a way to create the users its tokens
// belong to. Writes through WriteRepository must be visible through ReadRepository.
type ContractSubject struct {
	WriteRepository refreshtoken.WriteRepository
	ReadRepository  refreshtoken.ReadRepository
	SeedUser        func(t *testing.T) uuid.UUID
}

// RunRefreshTokenRepositoryContractParams builds a fresh ContractSubject for every contract case.
type RunRefreshTokenRepositoryContractParams struct {
	NewSubject func(t *testing.T) ContractSubject
}

// RunRefreshTokenRepositoryContract runs the shared cases, so the in-memory fake cannot drift from the SQL.
func RunRefreshTokenRepositoryContract(t *testing.T, params RunRefreshTokenRepositoryContractParams) {
	cases := []struct {
		name string
		run  func(t *testing.T, harness contractHarness)
	}{
		{name: "saved token is found by its hash", run: savedTokenIsFoundByItsHash},
		{name: "revoked token is no longer found nor revocable", run: revokedTokenIsNoLongerFoundNorRevocable},
		{name: "expired token is not found", run: expiredTokenIsNotFound},
		{name: "deleting a user's tokens leaves other users' tokens", run: deletingAUsersTokensLeavesOtherUsersTokens},
		{name: "unused deletion purges expired tokens and keeps recent ones", run: unusedDeletionPurgesExpiredTokens},
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

type saveForContractParams struct {
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

func (h contractHarness) save(t *testing.T, params saveForContractParams) refreshtoken.RefreshToken {
	t.Helper()

	saved, err := h.subject.WriteRepository.SaveRefreshToken(h.ctx, refreshtoken.NewRefreshToken{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: params.TokenHash,
		ExpiresAt: params.ExpiresAt,
	})
	if err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}

	return saved
}

func (h contractHarness) saveLive(t *testing.T, tokenHash string) refreshtoken.RefreshToken {
	t.Helper()

	return h.save(t, saveForContractParams{UserID: h.userID, TokenHash: tokenHash, ExpiresAt: time.Now().Add(time.Hour)})
}

func (h contractHarness) assertNotFound(t *testing.T, tokenHash string) {
	t.Helper()

	_, err := h.subject.ReadRepository.FindByTokenHash(h.ctx, tokenHash)
	if _, notFound := errors.AsType[refreshtoken.ErrRefreshTokenNotFound](err); !notFound {
		t.Errorf("FindByTokenHash(%q) = %v, want ErrRefreshTokenNotFound", tokenHash, err)
	}
}

func (h contractHarness) assertFound(t *testing.T, tokenHash string) {
	t.Helper()

	found, err := h.subject.ReadRepository.FindByTokenHash(h.ctx, tokenHash)
	if err != nil || found.TokenHash != tokenHash {
		t.Errorf("FindByTokenHash(%q) = %+v, %v, want the live token", tokenHash, found, err)
	}
}

func savedTokenIsFoundByItsHash(t *testing.T, harness contractHarness) {
	saved := harness.saveLive(t, "live-hash")

	found, err := harness.subject.ReadRepository.FindByTokenHash(harness.ctx, "live-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if found.ID != saved.ID || found.UserID != harness.userID || found.IsRevoked() {
		t.Errorf("found = %+v, want the unrevoked token %s", found, saved.ID)
	}
	harness.assertNotFound(t, "unknown-hash")
}

func revokedTokenIsNoLongerFoundNorRevocable(t *testing.T, harness contractHarness) {
	harness.saveLive(t, "revoked-hash")

	revoked, err := harness.subject.WriteRepository.Revoke(harness.ctx, "revoked-hash")
	if err != nil || !revoked.IsRevoked() {
		t.Fatalf("Revoke = %+v, %v, want the token stamped revoked", revoked, err)
	}

	harness.assertNotFound(t, "revoked-hash")
	_, err = harness.subject.WriteRepository.Revoke(harness.ctx, "revoked-hash")
	if _, notFound := errors.AsType[refreshtoken.ErrRefreshTokenNotFound](err); !notFound {
		t.Errorf("second Revoke = %v, want ErrRefreshTokenNotFound", err)
	}
}

func expiredTokenIsNotFound(t *testing.T, harness contractHarness) {
	harness.save(t, saveForContractParams{
		UserID:    harness.userID,
		TokenHash: "expired-hash",
		ExpiresAt: time.Now().Add(-time.Minute),
	})

	harness.assertNotFound(t, "expired-hash")
}

func deletingAUsersTokensLeavesOtherUsersTokens(t *testing.T, harness contractHarness) {
	otherUserID := harness.subject.SeedUser(t)
	harness.saveLive(t, "owner-first")
	harness.saveLive(t, "owner-second")
	harness.save(t, saveForContractParams{UserID: otherUserID, TokenHash: "other-user", ExpiresAt: time.Now().Add(time.Hour)})

	if err := harness.subject.WriteRepository.DeleteAllForUser(harness.ctx, harness.userID); err != nil {
		t.Fatalf("DeleteAllForUser: %v", err)
	}

	harness.assertNotFound(t, "owner-first")
	harness.assertNotFound(t, "owner-second")
	harness.assertFound(t, "other-user")
}

func unusedDeletionPurgesExpiredTokens(t *testing.T, harness contractHarness) {
	harness.saveLive(t, "live")
	harness.save(t, saveForContractParams{UserID: harness.userID, TokenHash: "expired", ExpiresAt: time.Now().Add(-time.Minute)})
	harness.saveLive(t, "recently-revoked")
	if _, err := harness.subject.WriteRepository.Revoke(harness.ctx, "recently-revoked"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	deleted, err := harness.subject.WriteRepository.DeleteUnused(harness.ctx, refreshtoken.CleanUpThresholdInDays)
	if err != nil {
		t.Fatalf("DeleteUnused: %v", err)
	}

	if deleted != 1 {
		t.Errorf("deleted = %d, want only the expired token purged", deleted)
	}
	harness.assertFound(t, "live")
}
