package config

import "fmt"

const failedIntParseText = "failed parsing int variable"
const portExpectationText = "expected an integer between 1 and 65535"
const envFileMalformedText = "malformed .env file"
const databaseURLExpectationText = "expected a postgres:// or postgresql:// connection URL, such as postgresql://user:password@localhost:5432/web_app_starter"
const rabbitMQURLExpectationText = "expected an amqp:// or amqps:// connection URL, such as amqp://user:password@localhost:5672/"
const jwtSecretExpectationText = "expected a jwt secret to be provided"
const frontendBaseURLExpectationText = "expected the base URL the frontend is served from, such as http://localhost:3001"
const appEnvExpectationText = "expected the app's environment to be set to local, dev or prod"
const brevoApiKeyExpectationText = "expected a Brevo API key, such as xkeysib-..."
const brevoSenderEmailExpectationText = "expected the address transactional emails are sent from, such as no-reply@example.com"
const brevoTemplateIDExpectationText = "expected the integer ID of the Brevo email verification template"
const firebaseCredentialsExpectationText = "expected the Firebase service account JSON encoded as base64"
const firebaseStorageBucketExpectationText = "expected the Firebase storage bucket name, such as my-project.firebasestorage.app"

// ErrFailedIntParse reports an environment variable that was set but is not a number.
type ErrFailedIntParse struct {
	Variable string
	RawValue string
}

func (e ErrFailedIntParse) Error() string {
	return fmt.Sprintf("invalid %s %q: %s", e.Variable, e.RawValue, failedIntParseText)
}

// ErrInvalidPort reports a port variable outside the 1..65535 range.
type ErrInvalidPort struct {
	Variable string
	RawValue string
}

func (e ErrInvalidPort) Error() string {
	return fmt.Sprintf("invalid %s %q: %s", e.Variable, e.RawValue, portExpectationText)
}

// ErrEnvFileMalformed reports a .env file that exists but could not be parsed.
type ErrEnvFileMalformed struct {
	Cause error
}

func (e ErrEnvFileMalformed) Error() string {
	return fmt.Sprintf("%s: %s", envFileMalformedText, e.Cause)
}

func (e ErrEnvFileMalformed) Unwrap() error {
	return e.Cause
}

// ErrMissingRequiredVariable reports a required environment variable that is unset or empty.
type ErrMissingRequiredVariable struct {
	Variable    string
	Expectation string
}

func (e ErrMissingRequiredVariable) Error() string {
	return fmt.Sprintf("missing %s: %s", e.Variable, e.Expectation)
}
