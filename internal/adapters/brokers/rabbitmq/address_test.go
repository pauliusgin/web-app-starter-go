package rabbitmq

import (
	"strings"
	"testing"
)

func TestBrokerAddressDropsCredentials(t *testing.T) {
	address := SanitizeBrokerAddress("amqp://user:s3cret@localhost:5672/web_app_starter")

	if strings.Contains(address, "s3cret") || strings.Contains(address, "user") {
		t.Fatalf("expected the credentials to be dropped, got %q", address)
	}
}

func TestBrokerAddress(t *testing.T) {
	testCases := []struct {
		name    string
		amqpURL string
		want    string
	}{
		{
			name:    "keeps the host and the vhost",
			amqpURL: "amqp://user:password@localhost:5672/web_app_starter",
			want:    "localhost:5672/web_app_starter",
		},
		{
			name:    "reports the default vhost when the URL carries no path",
			amqpURL: "amqp://user:password@localhost:5672",
			want:    "localhost:5672/",
		},
		{
			name:    "reports an unparseable URL as unknown",
			amqpURL: "amqp://user:password@localhost:56 72/web_app_starter",
			want:    unknownBrokerAddress,
		},
		{
			name:    "reports a URL without a host as unknown",
			amqpURL: "web_app_starter",
			want:    unknownBrokerAddress,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := SanitizeBrokerAddress(testCase.amqpURL)

			if got != testCase.want {
				t.Fatalf("expected %q, got %q", testCase.want, got)
			}
		})
	}
}
