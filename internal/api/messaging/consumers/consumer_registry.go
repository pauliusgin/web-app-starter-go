package consumers

import (
	"log/slog"

	"github.com/pauliusgin/web_app_starter-be/internal/api/messaging/handlers/messaging"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

// AuthChallengeEmailQueue is this service's own durable queue for email auth challenges.
const AuthChallengeEmailQueue = "web_app_starter.auth_challenge.email"

const consumerSkippedText = "event consumer skipped: its handler could not be built"

// Registry is every event this service consumes.
type Registry []Consumer

// RegistryDependencies carries the collaborators the registry hands to its handlers.
type RegistryDependencies struct {
	VerificationEmail messaging.VerificationEmailDependencies
	Logger            *slog.Logger
}

// BuildRegistry returns every event this service consumes and what each one runs.
func BuildRegistry(d RegistryDependencies) Registry {
	return validConsumers(validConsumersParams{
		Logger: d.Logger,
		Declared: Registry{
			NewConsumer(NewConsumerParams[auth.AuthChallengeGeneratedData]{
				Label:  events.AuthChallengeGeneratedLabel,
				Queue:  AuthChallengeEmailQueue,
				Handle: messaging.VerificationEmailOnChallenge(d.VerificationEmail),
			}),
		},
	})
}

type validConsumersParams struct {
	Declared Registry
	Logger   *slog.Logger
}

func validConsumers(params validConsumersParams) Registry {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	valid := Registry{}

	for _, declared := range params.Declared {
		if declared.Handle == nil {
			logger.Warn(consumerSkippedText, "queue", declared.Queue, "event", declared.Label)
			continue
		}

		valid = append(valid, declared)
	}

	return valid
}
