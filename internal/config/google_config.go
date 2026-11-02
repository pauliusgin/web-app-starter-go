package config

const (
	// GoogleClientIDVariable names the env var holding the OAuth client ID Sign in with Google tokens are issued for.
	GoogleClientIDVariable = "GOOGLE_CLIENT_ID"
)

// GoogleConfig is the google OAuth client this service accepts Sign in with Google ID tokens for.
type GoogleConfig struct {
	ClientID string
}

// EnvVariables is every env variable this config is read from.
func (GoogleConfig) EnvVariables() []string {
	return []string{GoogleClientIDVariable}
}

func loadGoogleConfig() (GoogleConfig, error) {
	clientID, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    GoogleClientIDVariable,
		Expectation: googleClientIDExpectationText,
	})
	if err != nil {
		return GoogleConfig{}, err
	}

	return GoogleConfig{ClientID: clientID}, nil
}
