package rabbitmq

import "net/url"

const unknownBrokerAddress = "unknown"

const defaultVHostPath = "/"

// BrokerAddress returns the broker's host and vhost, or "unknown" when the URL will not parse.
func SanitizeBrokerAddress(amqpURL string) string {
	parsed, err := url.Parse(amqpURL)
	if err != nil || parsed.Host == "" {
		return unknownBrokerAddress
	}

	vhostPath := parsed.EscapedPath()
	if vhostPath == "" {
		vhostPath = defaultVHostPath
	}

	return parsed.Host + vhostPath
}
