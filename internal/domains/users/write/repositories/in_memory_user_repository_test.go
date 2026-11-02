package repositories_test

import (
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories/userrepositorytest"
)

func TestInMemoryUserRepositoryHonoursTheContract(t *testing.T) {
	userrepositorytest.RunUserRepositoryContract(t, userrepositorytest.RunUserRepositoryContractParams{
		NewRepository: func(*testing.T) repositories.UserRepository {
			return repositories.NewInMemoryUserRepository()
		},
	})
}
