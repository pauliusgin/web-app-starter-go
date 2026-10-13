package main

import (
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
)

func TestBuildRouterDependencies(t *testing.T) {
	cases := []struct {
		name      string
		jwtSecret string
		wantErr   bool
	}{
		{name: "valid secret", jwtSecret: testJWTSecret},
		{name: "empty secret", jwtSecret: "", wantErr: true},
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

			if testCase.wantErr {
				if err == nil {
					t.Fatal("buildRouterDependencies = nil error, want an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("buildRouterDependencies = %v, want nil", err)
			}
		})
	}
}
