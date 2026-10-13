package config

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"
)

const dotEnvFileName = ".env"

const validDatabaseURL = "postgresql://web_app_starter:s3cret@localhost:5432/web_app_starter"

const validRabbitMQURL = "amqp://web_app_starter:s3cret@localhost:5672/"

const validJWTSecret = "0hVJ0Qm3kq1sQnQpJ5uHq2Rf8xWjT4lZ6bN9cA7dE1g="

const appEnv = "local"

const validFrontendBaseURL = "http://localhost:3001"

const validBrevoApiKey = "xkeysib-0123456789abcdef"

const validBrevoSenderEmail = "no-reply@web_app_starter.example"

const validBrevoTemplateID = 7

const validFirebaseCredentials = "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50In0="

const validFirebaseStorageBucket = "web_app_starter.firebasestorage.app"

const validDotEnvExtraLines = FrontendBaseURLVariable + "=" + validFrontendBaseURL + "\n" +
	BrevoApiKeyVariable + "=" + validBrevoApiKey + "\n" +
	BrevoSenderEmailVariable + "=" + validBrevoSenderEmail + "\n" +
	BrevoEmailVerificationTemplateIDVariable + "=7\n" +
	FirebaseCredentialsVariable + "=" + validFirebaseCredentials + "\n" +
	FirebaseStorageBucketVariable + "=" + validFirebaseStorageBucket + "\n"

func prepareLoaderEnvironment(t *testing.T) {
	t.Helper()

	t.Chdir(t.TempDir())

	variables := []string{
		ServerHostVariable,
		ServerPortVariable,
		DatabaseURLVariable,
		RabbitMQURLVariable,
		JWTSecretVariable,
		FrontendBaseURLVariable,
		BrevoApiKeyVariable,
		BrevoSenderEmailVariable,
		BrevoEmailVerificationTemplateIDVariable,
		FirebaseCredentialsVariable,
		FirebaseStorageBucketVariable,
		AppEnvVariable,
	}

	for _, variable := range variables {
		t.Setenv(variable, "")
		if err := os.Unsetenv(variable); err != nil {
			t.Fatalf("unsetting %s: %v", variable, err)
		}
	}

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

func setValidRequiredVariables(t *testing.T) {
	t.Helper()

	t.Setenv(DatabaseURLVariable, validDatabaseURL)
	t.Setenv(RabbitMQURLVariable, validRabbitMQURL)
	t.Setenv(JWTSecretVariable, validJWTSecret)
	t.Setenv(AppEnvVariable, appEnv)
	t.Setenv(FrontendBaseURLVariable, validFrontendBaseURL)
	t.Setenv(BrevoApiKeyVariable, validBrevoApiKey)
	t.Setenv(BrevoSenderEmailVariable, validBrevoSenderEmail)
	t.Setenv(BrevoEmailVerificationTemplateIDVariable, strconv.Itoa(validBrevoTemplateID))
	t.Setenv(FirebaseCredentialsVariable, validFirebaseCredentials)
	t.Setenv(FirebaseStorageBucketVariable, validFirebaseStorageBucket)
}

func writeDotEnvFile(t *testing.T, contents string) {
	t.Helper()

	if err := os.WriteFile(dotEnvFileName, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing %s: %v", dotEnvFileName, err)
	}
}

// net/http reads "0" as "no timeout", and context.WithTimeout(ctx, 0) is already expired
func TestServerTimeoutsAreUsable(t *testing.T) {
	cases := []struct {
		name    string
		timeout time.Duration
	}{
		{name: "read header", timeout: ReadHeaderTimeout},
		{name: "read", timeout: ReadTimeout},
		{name: "write", timeout: WriteTimeout},
		{name: "idle", timeout: IdleTimeout},
		{name: "shutdown", timeout: ShutdownTimeout},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.timeout <= 0 {
				t.Errorf("timeout = %s, want a positive duration", testCase.timeout)
			}
		})
	}

	if ReadHeaderTimeout > ReadTimeout {
		t.Errorf("ReadHeaderTimeout = %s, want no more than ReadTimeout = %s", ReadHeaderTimeout, ReadTimeout)
	}
}

func TestLoadConfigAppliesServerDefaults(t *testing.T) {
	prepareLoaderEnvironment(t)
	setValidRequiredVariables(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.ServerHost != DefaultServerHost {
		t.Errorf("ServerHost = %q, want %q", cfg.ServerHost, DefaultServerHost)
	}
	if cfg.ServerPort != DefaultServerPort {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, DefaultServerPort)
	}
	if cfg.ListenAddress() != "0.0.0.0:3000" {
		t.Errorf("ListenAddress() = %q, want %q", cfg.ListenAddress(), "0.0.0.0:3000")
	}
}

func TestLoadConfigReadsOverrides(t *testing.T) {
	prepareLoaderEnvironment(t)
	setValidRequiredVariables(t)
	t.Setenv(ServerHostVariable, "127.0.0.1")
	t.Setenv(ServerPortVariable, "9091")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	assertConfigMatchesOverrides(t, cfg)
}

func TestLoadConfigReadsSettingsFromDotEnvFile(t *testing.T) {
	prepareLoaderEnvironment(t)
	writeDotEnvFile(t, "APP_ENV=local\nSERVER_HOST=127.0.0.1\nSERVER_PORT=9091\nDATABASE_URL="+validDatabaseURL+"\nRABBITMQ_URL="+validRabbitMQURL+"\nJWT_SECRET="+validJWTSecret+"\n"+validDotEnvExtraLines)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	assertConfigMatchesOverrides(t, cfg)
}

func TestProcessEnvironmentWinsOverDotEnvFile(t *testing.T) {
	prepareLoaderEnvironment(t)
	writeDotEnvFile(t, "APP_ENV=local\nSERVER_HOST=127.0.0.1\nSERVER_PORT=4444\nDATABASE_URL="+validDatabaseURL+"\nRABBITMQ_URL="+validRabbitMQURL+"\nJWT_SECRET="+validJWTSecret+"\n"+validDotEnvExtraLines)
	t.Setenv(ServerPortVariable, "9091")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.ServerPort != 9091 {
		t.Errorf("ServerPort = %d, want %d (process environment must win over .env)", cfg.ServerPort, 9091)
	}
	if cfg.ServerHost != "127.0.0.1" {
		t.Errorf("ServerHost = %q, want %q (unset in the environment, so .env supplies it)", cfg.ServerHost, "127.0.0.1")
	}
}

func TestLoadConfigRequiresEveryRequiredVariable(t *testing.T) {
	cases := []struct {
		variable        string
		wantExpectation string
	}{
		{variable: DatabaseURLVariable, wantExpectation: databaseURLExpectationText},
		{variable: RabbitMQURLVariable, wantExpectation: rabbitMQURLExpectationText},
		{variable: JWTSecretVariable, wantExpectation: jwtSecretExpectationText},
		{variable: FrontendBaseURLVariable, wantExpectation: frontendBaseURLExpectationText},
		{variable: AppEnvVariable, wantExpectation: appEnvExpectationText},
		{variable: BrevoApiKeyVariable, wantExpectation: brevoApiKeyExpectationText},
		{variable: BrevoSenderEmailVariable, wantExpectation: brevoSenderEmailExpectationText},
		{variable: BrevoEmailVerificationTemplateIDVariable, wantExpectation: brevoTemplateIDExpectationText},
		{variable: FirebaseCredentialsVariable, wantExpectation: firebaseCredentialsExpectationText},
		{variable: FirebaseStorageBucketVariable, wantExpectation: firebaseStorageBucketExpectationText},
	}

	for _, testCase := range cases {
		t.Run(testCase.variable, func(t *testing.T) {
			prepareLoaderEnvironment(t)
			setValidRequiredVariables(t)
			unsetVariableForTest(t, testCase.variable)

			cfg, err := Load()

			missingVariable := ErrMissingRequiredVariable{}
			if !errors.As(err, &missingVariable) {
				t.Fatalf("error = %v, want ErrMissingRequiredVariable", err)
			}
			if missingVariable.Variable != testCase.variable {
				t.Errorf("Variable = %q, want %q", missingVariable.Variable, testCase.variable)
			}
			if missingVariable.Expectation != testCase.wantExpectation {
				t.Errorf("Expectation = %q, want %q", missingVariable.Expectation, testCase.wantExpectation)
			}
			if cfg != (Config{}) {
				t.Errorf("Config = %+v, want the zero value on failure", cfg)
			}
		})
	}
}

func TestLoadConfigReportsAnUnparsableDotEnvFile(t *testing.T) {
	prepareLoaderEnvironment(t)
	writeDotEnvFile(t, "SERVER-PORT=4444\n")

	_, err := Load()

	envFileMalformed := ErrEnvFileMalformed{}
	if !errors.As(err, &envFileMalformed) {
		t.Fatalf("error = %v, want ErrEnvFileMalformed", err)
	}
	if envFileMalformed.Cause == nil {
		t.Error("Cause = nil, want the parser error that says which line is wrong")
	}
}

func TestLoadConfigReportsEveryBadVariableAtOnce(t *testing.T) {
	prepareLoaderEnvironment(t)
	t.Setenv(ServerPortVariable, "eighty")

	cfg, err := Load()
	if err == nil {
		t.Fatal("LoadConfig = nil, want an error naming every bad variable")
	}
	if cfg != (Config{}) {
		t.Errorf("Config = %+v, want the zero value on failure", cfg)
	}

	reported := collectReportedVariables(err)
	wantReported := append([]string{
		ServerPortVariable,
		DatabaseURLVariable,
		RabbitMQURLVariable,
		JWTSecretVariable,
		FrontendBaseURLVariable,
	}, append((BrevoConfig{}).EnvVariables(), (FirebaseConfig{}).EnvVariables()...)...)
	for _, variable := range wantReported {
		if !slices.Contains(reported, variable) {
			t.Errorf("reported variables = %v, want %s among them", reported, variable)
		}
	}

	failedParse := ErrFailedIntParse{}
	if !errors.As(err, &failedParse) || failedParse.RawValue != "eighty" {
		t.Errorf("errors.As did not recover ErrFailedIntParse{RawValue: \"eighty\"} from %v", err)
	}

	missingVariable := ErrMissingRequiredVariable{}
	if !errors.As(err, &missingVariable) || missingVariable.Variable != DatabaseURLVariable {
		t.Errorf("errors.As did not recover the ErrMissingRequiredVariable for %s from %v", DatabaseURLVariable, err)
	}
}

func TestEnvFileMalformedUnwrapsToItsCause(t *testing.T) {
	cause := errors.New(`unexpected character "-" in variable name`)

	if !errors.Is(ErrEnvFileMalformed{Cause: cause}, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

// collectReportedVariables walks the errors.Join tree Load returns and names every variable it reports.
func collectReportedVariables(err error) []string {
	switch typed := err.(type) {
	case interface{ Unwrap() []error }:
		variables := []string{}
		for _, joined := range typed.Unwrap() {
			variables = append(variables, collectReportedVariables(joined)...)
		}
		return variables
	case ErrMissingRequiredVariable:
		return []string{typed.Variable}
	case ErrFailedIntParse:
		return []string{typed.Variable}
	case ErrInvalidPort:
		return []string{typed.Variable}
	}
	return nil
}

func assertConfigMatchesOverrides(t *testing.T, cfg Config) {
	t.Helper()

	if cfg.ServerHost != "127.0.0.1" {
		t.Errorf("ServerHost = %q, want %q", cfg.ServerHost, "127.0.0.1")
	}
	if cfg.ServerPort != 9091 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 9091)
	}
	if cfg.DatabaseURL != validDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, validDatabaseURL)
	}
	if cfg.RabbitMQURL != validRabbitMQURL {
		t.Errorf("RabbitMQURL = %q, want %q", cfg.RabbitMQURL, validRabbitMQURL)
	}
	if cfg.JWTSecret != validJWTSecret {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, validJWTSecret)
	}
	if cfg.FrontendBaseURL != validFrontendBaseURL {
		t.Errorf("FrontendBaseURL = %q, want %q", cfg.FrontendBaseURL, validFrontendBaseURL)
	}
	wantBrevo := BrevoConfig{
		ApiKey:                      validBrevoApiKey,
		SenderEmail:                 validBrevoSenderEmail,
		EmailVerificationTemplateID: validBrevoTemplateID,
	}
	if cfg.Brevo != wantBrevo {
		t.Errorf("Brevo = %+v, want %+v", cfg.Brevo, wantBrevo)
	}
	wantFirebase := FirebaseConfig{
		EncodedCredentials: validFirebaseCredentials,
		StorageBucket:      validFirebaseStorageBucket,
	}
	if cfg.Firebase != wantFirebase {
		t.Errorf("Firebase = %+v, want %+v", cfg.Firebase, wantFirebase)
	}
	if cfg.ListenAddress() != "127.0.0.1:9091" {
		t.Errorf("ListenAddress() = %q, want %q", cfg.ListenAddress(), "127.0.0.1:9091")
	}
}
