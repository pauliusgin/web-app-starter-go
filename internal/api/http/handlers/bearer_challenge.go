package handlers

// WWWAuthenticateHeader is the response header carrying the bearer challenge on a 401.
const WWWAuthenticateHeader = "WWW-Authenticate"

// InvalidTokenChallenge is the RFC 6750 challenge answering a token that is unknown, revoked or malformed.
const InvalidTokenChallenge = `Bearer error="invalid_token"`
