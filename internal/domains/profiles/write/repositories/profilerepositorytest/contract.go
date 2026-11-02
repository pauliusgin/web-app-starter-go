// Package profilerepositorytest holds the behaviour every profile repository pair must share.
package profilerepositorytest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
)

// ContractSubject is one fresh, empty store seen through both ports, and a way to create the users whose
// profiles it holds. Writes through WriteRepository must be visible through ReadRepository.
type ContractSubject struct {
	WriteRepository repositories.ProfileWriteRepository
	ReadRepository  read.ProfileReadRepository
	SeedUser        func(t *testing.T) uuid.UUID
}

// RunProfileRepositoryContractParams builds a fresh ContractSubject for every contract case.
type RunProfileRepositoryContractParams struct {
	NewSubject func(t *testing.T) ContractSubject
}

// RunProfileRepositoryContract runs the shared cases, so the in-memory fake cannot drift from the SQL.
func RunProfileRepositoryContract(t *testing.T, params RunProfileRepositoryContractParams) {
	cases := []struct {
		name string
		run  func(t *testing.T, harness contractHarness)
	}{
		{name: "saved profile is read back", run: savedProfileIsReadBack},
		{name: "a user holds at most one profile", run: aUserHoldsAtMostOneProfile},
		{name: "update replaces both names", run: updateReplacesBothNames},
		{name: "a user without a profile has none to read or update", run: aUserWithoutAProfileHasNone},
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

func (h contractHarness) save(t *testing.T) {
	t.Helper()

	_, err := h.subject.WriteRepository.SaveProfile(h.ctx, models.NewProfile{ID: h.userID, FirstName: "Ada", LastName: "Lovelace"})
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
}

func (h contractHarness) assertReadBack(t *testing.T, want readmodels.ProfileReadModel) {
	t.Helper()

	readBack, err := h.subject.ReadRepository.GetProfileReadModelByID(h.ctx, h.userID)
	if err != nil || readBack != want {
		t.Errorf("GetProfileReadModelByID = %+v, %v, want %+v", readBack, err, want)
	}
}

func assertProfileNotFound(t *testing.T, err error) {
	t.Helper()

	if _, notFound := errors.AsType[e.ErrProfileNotFound](err); !notFound {
		t.Errorf("error = %v, want ErrProfileNotFound", err)
	}
}

func savedProfileIsReadBack(t *testing.T, harness contractHarness) {
	harness.save(t)

	harness.assertReadBack(t, readmodels.ProfileReadModel{ID: harness.userID, FirstName: "Ada", LastName: "Lovelace"})
}

func aUserHoldsAtMostOneProfile(t *testing.T, harness contractHarness) {
	harness.save(t)

	_, err := harness.subject.WriteRepository.SaveProfile(harness.ctx, models.NewProfile{
		ID: harness.userID, FirstName: "Augusta", LastName: "King",
	})
	if _, exists := errors.AsType[e.ErrProfileAlreadyExists](err); !exists {
		t.Errorf("second SaveProfile = %v, want ErrProfileAlreadyExists", err)
	}
	harness.assertReadBack(t, readmodels.ProfileReadModel{ID: harness.userID, FirstName: "Ada", LastName: "Lovelace"})
}

func updateReplacesBothNames(t *testing.T, harness contractHarness) {
	harness.save(t)

	updated, err := harness.subject.WriteRepository.UpdateProfile(harness.ctx, models.UpdateProfileParams{
		ID: harness.userID, FirstName: "Augusta", LastName: "King",
	})
	if err != nil || updated.FirstName != "Augusta" || updated.LastName != "King" {
		t.Fatalf("UpdateProfile = %+v, %v, want both names replaced", updated, err)
	}
	harness.assertReadBack(t, readmodels.ProfileReadModel{ID: harness.userID, FirstName: "Augusta", LastName: "King"})
}

func aUserWithoutAProfileHasNone(t *testing.T, harness contractHarness) {
	_, err := harness.subject.ReadRepository.GetProfileReadModelByID(harness.ctx, harness.userID)
	assertProfileNotFound(t, err)

	_, err = harness.subject.WriteRepository.UpdateProfile(harness.ctx, models.UpdateProfileParams{
		ID: harness.userID, FirstName: "Augusta", LastName: "King",
	})
	assertProfileNotFound(t, err)
}
