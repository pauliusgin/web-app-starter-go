package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

// RefreshTokenMapper maps a stored row onto the domain type.
func RefreshTokenMapper(row sqlc.RefreshToken) refreshtoken.RefreshToken {
	mapped := refreshtoken.RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		CreatedAt: row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: row.RevokedAt.Time,
	}

	return mapped
}
