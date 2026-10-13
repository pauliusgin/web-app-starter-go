package main

import (
	"errors"
	"slices"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/config"
)

func TestBuildEmailGatewayRejectsAnUnconfiguredBrevoAccount(t *testing.T) {
	silenceDefaultLogger(t)

	cases := []struct {
		name    string
		account config.BrevoConfig
	}{
		{
			name: "placeholders left by config",
			account: config.BrevoConfig{
				ApiKey:                      "",
				SenderEmail:                 "",
				EmailVerificationTemplateID: 0,
			},
		},
		{
			name: "template id not provided",
			account: config.BrevoConfig{
				ApiKey:                      "test-api-key",
				SenderEmail:                 "no-reply@example.com",
				EmailVerificationTemplateID: 0,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			emailGateway, err := buildEmailGateway(testCase.account)

			var unavailable ErrEmailGatewayUnavailable
			if !errors.As(err, &unavailable) {
				t.Fatalf("buildEmailGateway = %v, want ErrEmailGatewayUnavailable", err)
			}
			if !slices.Equal(unavailable.Variables, config.BrevoConfig{}.EnvVariables()) {
				t.Errorf("Variables = %v, want the Brevo env variables", unavailable.Variables)
			}
			if emailGateway != nil {
				t.Error("a gateway was returned alongside the error, want none")
			}
		})
	}
}

func TestBuildFileStorageRejectsAnUnconfiguredFirebaseProject(t *testing.T) {
	cases := []struct {
		name    string
		project config.FirebaseConfig
	}{
		{name: "placeholders left by config", project: config.FirebaseConfig{}},
		{name: "credentials not base64", project: config.FirebaseConfig{
			EncodedCredentials: "{not base64}",
			StorageBucket:      "web_app_starter.firebasestorage.app",
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			resource, err := buildFileStorage(testCase.project)

			unavailable, ok := errors.AsType[ErrFileStorageUnavailable](err)
			if !ok {
				t.Fatalf("buildFileStorage = %v, want ErrFileStorageUnavailable", err)
			}
			if !slices.Equal(unavailable.Variables, config.FirebaseConfig{}.EnvVariables()) {
				t.Errorf("Variables = %v, want the Firebase env variables", unavailable.Variables)
			}
			if resource.FileStorage != nil {
				t.Error("a file storage was returned alongside the error, want none")
			}
		})
	}
}
