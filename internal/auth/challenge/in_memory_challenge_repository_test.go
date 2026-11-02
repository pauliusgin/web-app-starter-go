package challenge_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge/challengerepositorytest"
)

func TestInMemoryChallengeRepositoryHonoursTheContract(t *testing.T) {
	challengerepositorytest.RunChallengeRepositoryContract(t, challengerepositorytest.RunChallengeRepositoryContractParams{
		NewSubject: func(*testing.T) challengerepositorytest.ContractSubject {
			repository := challenge.NewInMemoryChallengeRepository()

			return challengerepositorytest.ContractSubject{
				WriteRepository: repository,
				ReadRepository:  repository,
				SeedUser:        func(*testing.T) uuid.UUID { return uuid.New() },
			}
		},
	})
}
