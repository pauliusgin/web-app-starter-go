package users

import "github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"

const testGoogleSubject = "109876543210987654321"

const verifiedGoogleIDToken = "verified-google-id-token"

const unverifiedGoogleIDToken = "unverified-google-id-token"

const verifiedGoogleIDTokenBody = `{"idToken":"` + verifiedGoogleIDToken + `"}`

func newGoogleVerifierForTest() googleidtoken.InMemoryVerifier {
	return googleidtoken.InMemoryVerifier{Identities: map[string]googleidtoken.Identity{
		verifiedGoogleIDToken:   {Subject: testGoogleSubject, Email: testEmail, EmailVerified: true},
		unverifiedGoogleIDToken: {Subject: "unverified-subject", Email: testEmail},
	}}
}
