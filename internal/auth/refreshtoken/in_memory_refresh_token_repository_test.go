package refreshtoken_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken/refreshtokenrepositorytest"
)

func TestInMemoryRefreshTokenRepositoryHonoursTheContract(t *testing.T) {
	refreshtokenrepositorytest.RunRefreshTokenRepositoryContract(t,
		refreshtokenrepositorytest.RunRefreshTokenRepositoryContractParams{
			NewSubject: func(*testing.T) refreshtokenrepositorytest.ContractSubject {
				repository := refreshtoken.NewInMemoryRefreshTokenRepository()

				return refreshtokenrepositorytest.ContractSubject{
					WriteRepository: repository,
					ReadRepository:  repository,
					SeedUser:        func(*testing.T) uuid.UUID { return uuid.New() },
				}
			},
		})
}
