package main

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
)

func TestBuildRouterDependencies(t *testing.T) {
	cases := []struct {
		name      string
		jwtSecret string
		wantErr   error
	}{
		{name: "valid secret", jwtSecret: testJWTSecret},
		{name: "empty secret", jwtSecret: "", wantErr: jwt.ErrEmptySecret{}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Building only wires closures; neither the database nor the broker is touched until a
			// request arrives, so a nil database and an unwired publisher are enough here.
			_, err := buildRouterDependencies(buildRouterDependenciesParams{
				Database:       nil,
				EventPublisher: rabbitmq.EventPublisher{},
				JWTSecret:      testCase.jwtSecret,
			})

			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("buildRouterDependencies = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
