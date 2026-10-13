package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/routes"
	"github.com/pauliusgin/web_app_starter-be/internal/config"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/scheduler"
)

// The defaults match the containers `make db-build` and `make mq-build` start; export
// DATABASE_URL / RABBITMQ_URL to point the live-infrastructure tests elsewhere.
const localDatabaseURL = "postgresql://user:password@localhost:5432/web_app_starter?sslmode=disable"

const localRabbitMQURL = "amqp://user:password@localhost:5672/web_app_starter"

const testJWTSecret = "0hVJ0Qm3kq1sQnQpJ5uHq2Rf8xWjT4lZ6bN9cA7dE1g="

const serverStartupTimeout = 10 * time.Second

var errStartupStepFailed = errors.New("startup step failed")

func silenceDefaultLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

type liveInfrastructure struct {
	DatabaseURL string
	RabbitMQURL string
}

func localInfrastructure() liveInfrastructure {
	return liveInfrastructure{
		DatabaseURL: readEnvOrDefault(config.DatabaseURLVariable, localDatabaseURL),
		RabbitMQURL: readEnvOrDefault(config.RabbitMQURLVariable, localRabbitMQURL),
	}
}

func readEnvOrDefault(variable string, fallback string) string {
	value := os.Getenv(variable)
	if value == "" {
		return fallback
	}
	return value
}

// requireLiveInfrastructure skips rather than fails when Postgres or RabbitMQ is down, because
// runServer cannot get past its database and broker s without them.
func requireLiveInfrastructure(t *testing.T) {
	t.Helper()

	infrastructure := localInfrastructure()

	db, err := postgres.NewDatabaseConnection(context.Background(), infrastructure.DatabaseURL)
	if err != nil {
		t.Skipf("postgres unavailable (run `make db-start`): %v", err)
	}
	db.Close()

	broker, err := openEventBroker(openEventBrokerParams{AmqpURL: infrastructure.RabbitMQURL})
	if err != nil {
		t.Skipf("rabbitmq unavailable (run `make mq-start`): %v", err)
	}
	broker.Close()
}

func reserveFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func serverAddress(port int) string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}

// testStartupSteps is the production wiring with config fixed, so a developer's .env or shell
// cannot change what the test runs against, and email, file storage and imaging stay in memory.
func testStartupSteps(serverPort int) startupSteps {
	infrastructure := localInfrastructure()

	s := serverStartupSteps()
	s.LoadConfig = func() (config.Config, error) {
		return config.Config{
			ServerHost:  "127.0.0.1",
			ServerPort:  serverPort,
			DatabaseURL: infrastructure.DatabaseURL,
			RabbitMQURL: infrastructure.RabbitMQURL,
			JWTSecret:   testJWTSecret,
		}, nil
	}
	s.BuildEmailGateway = func(config.BrevoConfig) (email.Gateway, error) {
		return email.NewInMemoryGateway(), nil
	}
	s.BuildFileStorage = func(config.FirebaseConfig) (fileStorageResource, error) {
		return fileStorageResource{FileStorage: storage.NewInMemoryFileStorage(), Close: func() {}}, nil
	}
	s.BuildImageProcessor = func() (imageProcessorResource, error) {
		return imageProcessorResource{
			ImageProcessor: imaging.NewInMemoryImageProcessor(imaging.ImageInfo{Width: 800, Height: 600}),
			Close:          func() {},
		}, nil
	}

	return s
}

func runServerInBackground(s startupSteps) <-chan error {
	result := make(chan error, 1)
	go func() { result <- runServer(s) }()
	return result
}

type awaitHealthyServerParams struct {
	Port         int
	ServerResult <-chan error
}

// awaitHealthyServer polls the health route; a 200 also proves the signal watch is registered,
// since runServer installs it before it starts serving.
func awaitHealthyServer(t *testing.T, params awaitHealthyServerParams) {
	t.Helper()

	healthURL := "http://" + serverAddress(params.Port) + routes.HealthPath
	deadline := time.Now().Add(serverStartupTimeout)

	for time.Now().Before(deadline) {
		select {
		case err := <-params.ServerResult:
			t.Fatalf("runServer = %v before it became healthy, want it serving", err)
		default:
		}

		response, err := http.Get(healthURL)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("GET %s did not answer 200 within %s", healthURL, serverStartupTimeout)
}

// interruptServer delivers the SIGINT a terminal Ctrl+C would; runServer has registered for it
// by the time the server answers health checks, so the test process itself is not killed.
func interruptServer(t *testing.T, result <-chan error) error {
	t.Helper()

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("sending SIGINT: %v", err)
	}

	select {
	case err := <-result:
		return err
	case <-time.After(config.ShutdownTimeout):
		t.Fatalf("runServer did not return within %s of SIGINT", config.ShutdownTimeout)
		return nil
	}
}

func TestRunServerServesWithoutError(t *testing.T) {
	silenceDefaultLogger(t)
	requireLiveInfrastructure(t)

	port := reserveFreePort(t)
	result := runServerInBackground(testStartupSteps(port))

	awaitHealthyServer(t, awaitHealthyServerParams{Port: port, ServerResult: result})

	select {
	case err := <-result:
		t.Fatalf("runServer = %v while serving, want it still running", err)
	default:
	}

	interruptServer(t, result)
}

func TestRunServerReportsFailedStartupStep(t *testing.T) {
	cases := []struct {
		name                    string
		needsLiveInfrastructure bool
		breakStep               func(s *startupSteps)
	}{
		{
			name: "config load",
			breakStep: func(s *startupSteps) {
				s.LoadConfig = func() (config.Config, error) {
					return config.Config{}, errStartupStepFailed
				}
			},
		},
		{
			name: "email gateway",
			breakStep: func(s *startupSteps) {
				s.BuildEmailGateway = func(config.BrevoConfig) (email.Gateway, error) {
					return nil, errStartupStepFailed
				}
			},
		},
		{
			name: "file storage",
			breakStep: func(s *startupSteps) {
				s.BuildFileStorage = func(config.FirebaseConfig) (fileStorageResource, error) {
					return fileStorageResource{}, errStartupStepFailed
				}
			},
		},
		{
			name: "image processor",
			breakStep: func(s *startupSteps) {
				s.BuildImageProcessor = func() (imageProcessorResource, error) {
					return imageProcessorResource{}, errStartupStepFailed
				}
			},
		},
		{
			name: "database connection",
			breakStep: func(s *startupSteps) {
				s.ConnectDatabase = func(context.Context, string) (*sql.DB, error) {
					return nil, errStartupStepFailed
				}
			},
		},
		{
			name:                    "event broker",
			needsLiveInfrastructure: true,
			breakStep: func(s *startupSteps) {
				s.OpenEventBroker = func(openEventBrokerParams) (eventBroker, error) {
					return eventBroker{}, errStartupStepFailed
				}
			},
		},
		{
			name:                    "event consumers",
			needsLiveInfrastructure: true,
			breakStep: func(s *startupSteps) {
				s.StartEventConsumers = func(startEventConsumersParams) (func(), error) {
					return nil, errStartupStepFailed
				}
			},
		},
		{
			name:                    "scheduler",
			needsLiveInfrastructure: true,
			breakStep: func(s *startupSteps) {
				s.BuildScheduler = func(schedulerParams) (scheduler.Scheduler, error) {
					return scheduler.Scheduler{}, errStartupStepFailed
				}
			},
		},
		{
			name:                    "router dependencies",
			needsLiveInfrastructure: true,
			breakStep: func(s *startupSteps) {
				s.BuildRouterDependencies = func(
					buildRouterDependenciesParams,
				) (routes.RouterDependencies, error) {
					return routes.RouterDependencies{}, errStartupStepFailed
				}
			},
		},
		{
			name:                    "tcp listen",
			needsLiveInfrastructure: true,
			breakStep: func(s *startupSteps) {
				s.ListenTCP = func(string, string) (net.Listener, error) {
					return nil, errStartupStepFailed
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceDefaultLogger(t)
			if testCase.needsLiveInfrastructure {
				requireLiveInfrastructure(t)
			}

			s := testStartupSteps(reserveFreePort(t))
			testCase.breakStep(&s)

			if err := runServer(s); err == nil {
				t.Fatal("runServer = nil, want an error")
			}
		})
	}
}

func TestRunServerRefusesToStartWithoutFirebaseConfig(t *testing.T) {
	silenceDefaultLogger(t)

	s := testStartupSteps(reserveFreePort(t))
	s.BuildFileStorage = buildFileStorage

	err := runServer(s)

	if _, ok := errors.AsType[ErrFileStorageUnavailable](err); !ok {
		t.Fatalf("runServer = %v, want ErrFileStorageUnavailable", err)
	}
}

func TestRunServerRefusesToStartWithoutBrevoConfig(t *testing.T) {
	silenceDefaultLogger(t)

	s := testStartupSteps(reserveFreePort(t))
	s.BuildEmailGateway = buildEmailGateway

	if err := runServer(s); err == nil {
		t.Fatal("runServer = nil, want an error")
	}
}

func TestRunServerShutsDownGracefullyOnInterrupt(t *testing.T) {
	silenceDefaultLogger(t)
	requireLiveInfrastructure(t)

	port := reserveFreePort(t)
	result := runServerInBackground(testStartupSteps(port))

	awaitHealthyServer(t, awaitHealthyServerParams{Port: port, ServerResult: result})

	if err := interruptServer(t, result); err != nil {
		t.Fatalf("runServer = %v, want nil after a graceful shutdown", err)
	}

	if _, err := net.DialTimeout("tcp", serverAddress(port), time.Second); err == nil {
		t.Error("port still accepts connections after shutdown, want the listener closed")
	}
}
