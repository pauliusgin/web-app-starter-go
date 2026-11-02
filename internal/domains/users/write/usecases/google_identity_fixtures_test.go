package usecases

import "github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"

const verifiedGoogleIDToken = "verified-google-id-token"

const unverifiedGoogleIDToken = "unverified-google-id-token"

// newGoogleVerifier vouches for testEmail under testGoogleSubject, and for an unverified email beside it.
func newGoogleVerifier() googleidtoken.InMemoryVerifier {
	return googleidtoken.InMemoryVerifier{Identities: map[string]googleidtoken.Identity{
		verifiedGoogleIDToken:   {Subject: testGoogleSubject, Email: " Accountant@Example.com ", EmailVerified: true},
		unverifiedGoogleIDToken: {Subject: "unverified-subject", Email: "stranger@example.com"},
	}}
}
