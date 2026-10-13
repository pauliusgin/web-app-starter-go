// Package config loads and validates the process environment the server needs to start.
package config

import (
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	// ServerHostVariable names the env var holding the address the HTTP server binds to.
	ServerHostVariable = "SERVER_HOST"
	// ServerPortVariable names the env var holding the port the HTTP server listens on.
	ServerPortVariable = "SERVER_PORT"
	// DatabaseURLVariable names the env var holding the Postgres connection URL.
	DatabaseURLVariable = "DATABASE_URL"
	// RabbitMQURLVariable names the env var holding the RabbitMQ connection URL.
	RabbitMQURLVariable = "RABBITMQ_URL"
	// JWTSecretVariable names the env var holding the secret access tokens are signed with.
	JWTSecretVariable = "JWT_SECRET"
	// FrontendBaseURLVariable names the env var holding the base URL the frontend is served from.
	FrontendBaseURLVariable = "FRONTEND_BASE_URL"
	// AppEnvVariable names the env var holding the app's environment.
	AppEnvVariable = "APP_ENV"
)

// DefaultServerHost is the host used when SERVER_HOST is unset.
const DefaultServerHost = "0.0.0.0"

// DefaultServerPort is the port used when SERVER_PORT is unset.
const DefaultServerPort = 3000

const (
	// ReadHeaderTimeout bounds how long the server waits for a request's headers.
	ReadHeaderTimeout = 5 * time.Second
	// ReadTimeout bounds how long the server waits for a full request to arrive.
	ReadTimeout = 60 * time.Second
	// WriteTimeout bounds how long a handler has to write its response.
	WriteTimeout = 30 * time.Second
	// IdleTimeout bounds how long a keep-alive connection may sit unused.
	IdleTimeout = 60 * time.Second
	// ShutdownTimeout bounds how long in-flight requests have to finish on shutdown.
	ShutdownTimeout = 30 * time.Second
	// SchedulerStopTimeout bounds how long a mid-run background job has to finish on shutdown.
	SchedulerStopTimeout = 30 * time.Second
	// ConsumerStopTimeout bounds how long a mid-flight event handler has to finish on shutdown.
	ConsumerStopTimeout = 30 * time.Second
)

// Config is the validated environment the server was started with.
type Config struct {
	ServerHost      string
	ServerPort      int
	DatabaseURL     string
	RabbitMQURL     string
	JWTSecret       string
	FrontendBaseURL string
	Brevo           BrevoConfig
	Firebase        FirebaseConfig
	AppEnv          string
}

// ListenAddress is the host:port string the HTTP server binds to.
func (c Config) ListenAddress() string {
	return net.JoinHostPort(c.ServerHost, strconv.Itoa(c.ServerPort))
}

// Load reads .env and the environment, reporting all missing variables at once.
func Load() (Config, error) {
	var failures []error

	err := readEnvFile()
	failures = append(failures, err)

	serverHost := readStringVariable(readStringVariableParams{
		Variable: ServerHostVariable,
		Fallback: DefaultServerHost,
	})

	serverPort, err := readPortVariable(readIntVariableParams{
		Variable: ServerPortVariable,
		Fallback: DefaultServerPort,
	})
	failures = append(failures, err)

	databaseURL, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    DatabaseURLVariable,
		Expectation: databaseURLExpectationText,
	})
	failures = append(failures, err)

	rabbitMQURL, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    RabbitMQURLVariable,
		Expectation: rabbitMQURLExpectationText,
	})
	failures = append(failures, err)

	jwtSecret, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    JWTSecretVariable,
		Expectation: jwtSecretExpectationText,
	})
	failures = append(failures, err)

	brevo, err := loadBrevoConfig()
	failures = append(failures, err)

	firebase, err := loadFirebaseConfig()
	failures = append(failures, err)

	frontendBaseURL, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    FrontendBaseURLVariable,
		Expectation: frontendBaseURLExpectationText,
	})
	failures = append(failures, err)

	appEnv, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    AppEnvVariable,
		Expectation: appEnvExpectationText,
	})
	failures = append(failures, err)

	if joined := errors.Join(failures...); joined != nil {
		return Config{}, joined
	}

	return Config{
		ServerHost:      serverHost,
		ServerPort:      serverPort,
		DatabaseURL:     databaseURL,
		RabbitMQURL:     rabbitMQURL,
		JWTSecret:       jwtSecret,
		FrontendBaseURL: frontendBaseURL,
		Brevo:           brevo,
		Firebase:        firebase,
		AppEnv:          appEnv,
	}, nil
}

func readEnvFile() error {
	err := godotenv.Load()
	if err == nil {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		slog.Info("no .env file found, reading configuration from the process environment")
		return nil
	}
	return ErrEnvFileMalformed{Cause: err}
}
