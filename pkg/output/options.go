// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package output

import "time"

type Options struct {
	Addr            string        // Destination address (host:port).
	Delay           time.Duration // Delay start after start signal.
	Protocol        string        // Protocol (udp/tcp/tls).
	Retries         int           // Number of connection retries for tcp based protocols.
	StartSignal     string        // OS signal to wait on before starting.
	InsecureTLS     bool          // Disable TLS verification checks.
	RateLimit       int           // UDP rate limit in bytes.
	LogReaderBuffer int           // Log reader buffer size in bytes.

	WebhookOptions
	GCPPubsubOptions
	HTTPServerOptions
}

type WebhookOptions struct {
	ContentType string   // Content-Type header.
	Headers     []string // Headers in Key=Value format.
	Username    string   // Basic auth username.
	Password    string   // Basic auth password.
}

type GCPPubsubOptions struct {
	Project      string // Project name.
	Topic        string // Topic name. Will create it if not exists.
	Subscription string // Subscription name. Will create it if not exists.
	Clear        bool   // Clear will clear all topics and subscriptions before running.
}

type HTTPServerOptions struct {
	TLSCertificate  string        // TLS certificate file path.
	TLSKey          string        // TLS key file path.
	ResponseHeaders []string      // KV list of response headers.
	ReadTimeout     time.Duration // HTTP Server read timeout.
	WriteTimeout    time.Duration // HTTP Server write timeout.
}
