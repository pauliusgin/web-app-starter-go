package rabbitmq

import (
	"testing"
)

func TestSanitizeBrokerAddress(t *testing.T) {
	cases := []struct {
		name    string
		amqpURL string
		want    string
	}{
		{
			name:    "drops the credentials and keeps the host and the vhost",
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

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := SanitizeBrokerAddress(testCase.amqpURL)

			if got != testCase.want {
				t.Fatalf("expected %q, got %q", testCase.want, got)
			}
		})
	}
}
