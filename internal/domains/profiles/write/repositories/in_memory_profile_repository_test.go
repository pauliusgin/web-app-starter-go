package repositories_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories/profilerepositorytest"
)

func TestInMemoryProfileRepositoryHonoursTheContract(t *testing.T) {
	profilerepositorytest.RunProfileRepositoryContract(t, profilerepositorytest.RunProfileRepositoryContractParams{
		NewSubject: func(*testing.T) profilerepositorytest.ContractSubject {
			repository := repositories.NewInMemoryProfileRepository()

			return profilerepositorytest.ContractSubject{
				WriteRepository: repository,
				ReadRepository:  repository,
				SeedUser:        func(*testing.T) uuid.UUID { return uuid.New() },
			}
		},
	})
}
