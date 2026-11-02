package repositories_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories/uploadrepositorytest"
)

func TestInMemoryUploadRepositoryHonoursTheContract(t *testing.T) {
	uploadrepositorytest.RunUploadRepositoryContract(t, uploadrepositorytest.RunUploadRepositoryContractParams{
		NewSubject: func(*testing.T) uploadrepositorytest.ContractSubject {
			repository := repositories.NewInMemoryUploadRepository()

			return uploadrepositorytest.ContractSubject{
				WriteRepository: repository,
				ReadRepository:  repository,
				SeedUser:        func(*testing.T) uuid.UUID { return uuid.New() },
			}
		},
	})
}
