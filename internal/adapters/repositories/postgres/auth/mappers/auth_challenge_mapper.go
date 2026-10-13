// Package mappers converts stored database rows into domain types.
package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
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
