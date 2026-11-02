package integration

import (
	"testing"

	"github.com/google/uuid"

	authrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/repositories"
	profilesrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/profiles/repositories"
	uploadsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge/challengerepositorytest"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken/refreshtokenrepositorytest"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories/profilerepositorytest"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories/uploadrepositorytest"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories/userrepositorytest"
)

func TestPostgresUserRepositoryHonoursTheContract(t *testing.T) {
	userrepositorytest.RunUserRepositoryContract(t, userrepositorytest.RunUserRepositoryContractParams{
		NewRepository: func(t *testing.T) repositories.UserRepository {
			return newIntegrationWorld(t).userWriteRepository
		},
	})
}

func TestPostgresChallengeRepositoriesHonourTheContract(t *testing.T) {
	challengerepositorytest.RunChallengeRepositoryContract(t, challengerepositorytest.RunChallengeRepositoryContractParams{
		NewSubject: func(t *testing.T) challengerepositorytest.ContractSubject {
			world := newIntegrationWorld(t)

			return challengerepositorytest.ContractSubject{
				WriteRepository: authrepo.NewAuthChallengeWriteRepository(world.queries),
				ReadRepository:  world.challengeReadRepo,
				SeedUser: func(t *testing.T) uuid.UUID {
					return world.signUp(t, "owner-"+uuid.NewString()[:8]+"@example.com").User.ID
				},
			}
		},
	})
}

func TestPostgresRefreshTokenRepositoriesHonourTheContract(t *testing.T) {
	refreshtokenrepositorytest.RunRefreshTokenRepositoryContract(t,
		refreshtokenrepositorytest.RunRefreshTokenRepositoryContractParams{
			NewSubject: func(t *testing.T) refreshtokenrepositorytest.ContractSubject {
				world := newIntegrationWorld(t)

				return refreshtokenrepositorytest.ContractSubject{
					WriteRepository: world.refreshTokenWriteRepo,
					ReadRepository:  world.refreshTokenReadRepo,
					SeedUser: func(t *testing.T) uuid.UUID {
						return world.signUp(t, "owner-"+uuid.NewString()[:8]+"@example.com").User.ID
					},
				}
			},
		})
}

func TestPostgresUploadRepositoriesHonourTheContract(t *testing.T) {
	uploadrepositorytest.RunUploadRepositoryContract(t, uploadrepositorytest.RunUploadRepositoryContractParams{
		NewSubject: func(t *testing.T) uploadrepositorytest.ContractSubject {
			world := newIntegrationWorld(t)

			return uploadrepositorytest.ContractSubject{
				WriteRepository: world.uploadRepository,
				ReadRepository:  uploadsrepo.NewUploadReadRepository(world.queries),
				SeedUser: func(t *testing.T) uuid.UUID {
					return world.signUp(t, "owner-"+uuid.NewString()[:8]+"@example.com").User.ID
				},
				ReferenceUpload: world.referenceUploadAsProfilePicture,
			}
		},
	})
}

func TestPostgresProfileRepositoriesHonourTheContract(t *testing.T) {
	profilerepositorytest.RunProfileRepositoryContract(t, profilerepositorytest.RunProfileRepositoryContractParams{
		NewSubject: func(t *testing.T) profilerepositorytest.ContractSubject {
			world := newIntegrationWorld(t)

			return profilerepositorytest.ContractSubject{
				WriteRepository: world.profileRepository,
				ReadRepository:  profilesrepo.NewProfileReadRepository(world.queries),
				SeedUser: func(t *testing.T) uuid.UUID {
					return world.signUp(t, "owner-"+uuid.NewString()[:8]+"@example.com").User.ID
				},
			}
		},
	})
}
