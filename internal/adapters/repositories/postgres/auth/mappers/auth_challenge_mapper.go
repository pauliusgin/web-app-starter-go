// Package mappers converts stored database rows into domain types.
package mappers

import (
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

// AuthChallengeMapper maps a stored row onto the domain type.
func AuthChallengeMapper(row sqlc.AuthChallenge) challenge.Challenge {
	mapped := challenge.Challenge{
		ID:          row.ID,
		UserID:      row.UserID,
		TargetType:  challenge.TargetType(row.TargetType),
		Target:      row.Target,
		Purpose:     challenge.Purpose(row.Purpose),
		CreatedAt:   row.CreatedAt,
		ExpiresAt:   row.ExpiresAt,
		TokenHash:   row.TokenHash.String,
		Code:        row.Code.String,
		Provider:    row.Provider.String,
		ReferenceID: row.ReferenceID.String,
		UsedAt:      row.UsedAt.Time,
	}

	return mapped
}

// CreateAuthChallengeParamsMapper maps a new challenge onto the CreateAuthChallenge insert parameters.
func CreateAuthChallengeParamsMapper(
	newChallenge challenge.NewChallenge,
	createdAt time.Time,
) sqlc.CreateAuthChallengeParams {
	return sqlc.CreateAuthChallengeParams{
		ID:         newChallenge.ID,
		UserID:     newChallenge.UserID,
		TargetType: string(newChallenge.TargetType),
		Target:     newChallenge.Target,
		Purpose:    string(newChallenge.Purpose),
		TokenHash:  sqlnull.StringUnlessEmpty(newChallenge.TokenHash),
		CreatedAt:  createdAt,
		ExpiresAt:  newChallenge.ExpiresAt,
	}
}
