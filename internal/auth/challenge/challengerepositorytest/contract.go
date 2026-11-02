// Package challengerepositorytest holds the behaviour every auth challenge repository pair must share.
package challengerepositorytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

// ContractSubject is one fresh store seen through both ports, and a way to create the users its challenges
// belong to. Writes through WriteRepository must be visible through ReadRepository.
type ContractSubject struct {
	WriteRepository challenge.WriteRepository
	ReadRepository  challenge.ReadRepository
	SeedUser        func(t *testing.T) uuid.UUID
}

// RunChallengeRepositoryContractParams builds a fresh ContractSubject for every contract case.
type RunChallengeRepositoryContractParams struct {
	NewSubject func(t *testing.T) ContractSubject
}

// RunChallengeRepositoryContract runs the shared cases, so the in-memory fake cannot drift from the SQL.
func RunChallengeRepositoryContract(t *testing.T, params RunChallengeRepositoryContractParams) {
	cases := []struct {
		name string
		run  func(t *testing.T, harness contractHarness)
	}{
		{name: "saved challenge is found by its token hash", run: savedChallengeIsFoundByItsTokenHash},
		{name: "used challenge is no longer found", run: usedChallengeIsNoLongerFound},
		{name: "expired challenge is not found", run: expiredChallengeIsNotFound},
		{name: "marking a used challenge again is not an error", run: markingAUsedChallengeAgainIsNotAnError},
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
	TokenHash string
	Target    string
	Purpose   challenge.Purpose
	ExpiresAt time.Time
}

func (h contractHarness) save(t *testing.T, params saveForContractParams) challenge.Challenge {
	t.Helper()

	saved, err := h.subject.WriteRepository.SaveChallenge(h.ctx, challenge.NewChallenge{
		ID:         uuid.New(),
		UserID:     h.userID,
		TargetType: challenge.TargetTypeEmail,
		Target:     params.Target,
		Purpose:    params.Purpose,
		TokenHash:  params.TokenHash,
		ExpiresAt:  params.ExpiresAt,
	})
	if err != nil {
		t.Fatalf("SaveChallenge: %v", err)
	}

	return saved
}

func (h contractHarness) saveLive(t *testing.T, tokenHash string) challenge.Challenge {
	t.Helper()

	return h.save(t, saveForContractParams{
		TokenHash: tokenHash,
		Target:    "owner@example.com",
		Purpose:   challenge.PurposeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	})
}

func (h contractHarness) assertNotFound(t *testing.T, tokenHash string) {
	t.Helper()

	_, err := h.subject.ReadRepository.FindByTokenHash(h.ctx, tokenHash)
	if _, notFound := errors.AsType[challenge.ErrChallengeNotFound](err); !notFound {
		t.Errorf("FindByTokenHash = %v, want ErrChallengeNotFound", err)
	}
}

func savedChallengeIsFoundByItsTokenHash(t *testing.T, harness contractHarness) {
	saved := harness.saveLive(t, "live-hash")

	found, err := harness.subject.ReadRepository.FindByTokenHash(harness.ctx, "live-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if found.ID != saved.ID || found.UserID != harness.userID || found.IsUsed() {
		t.Errorf("found = %+v, want the unused challenge %s", found, saved.ID)
	}
	harness.assertNotFound(t, "unknown-hash")
}

func usedChallengeIsNoLongerFound(t *testing.T, harness contractHarness) {
	saved := harness.saveLive(t, "spent-hash")

	if err := harness.subject.WriteRepository.MarkUsed(harness.ctx, saved.ID); err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}

	harness.assertNotFound(t, "spent-hash")
}

func expiredChallengeIsNotFound(t *testing.T, harness contractHarness) {
	harness.save(t, saveForContractParams{
		TokenHash: "expired-hash",
		Target:    "owner@example.com",
		Purpose:   challenge.PurposeEmailVerification,
		ExpiresAt: time.Now().Add(-time.Minute),
	})

	harness.assertNotFound(t, "expired-hash")
}

func markingAUsedChallengeAgainIsNotAnError(t *testing.T, harness contractHarness) {
	saved := harness.saveLive(t, "twice-hash")

	for attempt := 1; attempt <= 2; attempt++ {
		if err := harness.subject.WriteRepository.MarkUsed(harness.ctx, saved.ID); err != nil {
			t.Fatalf("MarkUsed attempt %d: %v", attempt, err)
		}
	}
	if err := harness.subject.WriteRepository.MarkUsed(harness.ctx, uuid.New()); err != nil {
		t.Errorf("MarkUsed(unknown) = %v, want nil", err)
	}
}
