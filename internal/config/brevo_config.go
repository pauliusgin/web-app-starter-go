package config

import "errors"

const (
	// BrevoApiKeyVariable names the env var holding the API key for the Brevo API.
	BrevoApiKeyVariable = "BREVO_API_KEY"
	// BrevoSenderEmailVariable names the env var holding the address emails are sent from.
	BrevoSenderEmailVariable = "BREVO_SENDER_EMAIL"
	// BrevoEmailVerificationTemplateIDVariable names the env var holding the verification template.
	BrevoEmailVerificationTemplateIDVariable = "BREVO_EMAIL_VERIFICATION_TEMPLATE_ID"
)

// BrevoConfig is the Brevo account this service sends transactional email through.
type BrevoConfig struct {
	ApiKey                      string
	SenderEmail                 string
	EmailVerificationTemplateID int
}

// EnvVariables is every env variable this config is read from.
func (BrevoConfig) EnvVariables() []string {
	return []string{
		BrevoApiKeyVariable,
		BrevoSenderEmailVariable,
		BrevoEmailVerificationTemplateIDVariable,
	}
}

func loadBrevoConfig() (BrevoConfig, error) {
	apiKey, apiKeyErr := readRequiredVariable(readRequiredVariableParams{
		Variable:    BrevoApiKeyVariable,
		Expectation: brevoApiKeyExpectationText,
	})

	senderEmail, senderEmailErr := readRequiredVariable(readRequiredVariableParams{
		Variable:    BrevoSenderEmailVariable,
		Expectation: brevoSenderEmailExpectationText,
	})

	emailVerificationTemplateID, templateIDErr := readRequiredIntVariable(readRequiredVariableParams{
		Variable:    BrevoEmailVerificationTemplateIDVariable,
		Expectation: brevoTemplateIDExpectationText,
	})

	if err := errors.Join(apiKeyErr, senderEmailErr, templateIDErr); err != nil {
		return BrevoConfig{}, err
	}

	return BrevoConfig{
		ApiKey:                      apiKey,
		SenderEmail:                 senderEmail,
		EmailVerificationTemplateID: emailVerificationTemplateID,
	}, nil
}
