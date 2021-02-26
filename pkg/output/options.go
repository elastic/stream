// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package output

import "time"

type Options struct {
	Addr        string        // Destination address (host:port).
	Delay       time.Duration // Delay start after start signal.
	Protocol    string        // Protocol (udp/tcp/tls).
	Retries     int           // Number of connection retries for tcp based protocols.
	StartSignal string        // OS signal to wait on before starting.
	InsecureTLS bool          // Disable TLS verification checks.
	RateLimit   int           // UDP rate limit in bytes.

	WebhookOptions
}

type WebhookOptions struct {
	ContentType string   // Content-Type header.
	Headers     []string // Headers in Key=Value format.
	Username    string   // Basic auth username.
	Password    string   // Basic auth password.
}
