// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package output

import "time"

type Options struct {
	Addr           string        // Destination address (host:port).
	Delay          time.Duration // Delay start after start signal.
	Protocol       string        // Protocol (udp/tcp/tls).
	Retries        int           // Number of connection retries for tcp based protocols.
	StartSignal    string        // OS signal to wait on before starting.
	InsecureTLS    bool          // Disable TLS verification checks.
	RateLimit      int           // UDP rate limit in bytes.
	MaxLogLineSize int           // Log reader buffer size in bytes.

	WebhookOptions
	GCPPubsubOptions
	KafkaOptions
	AzureBlobStorageOptions
	LumberjackOptions
	GcsOptions
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

type KafkaOptions struct {
	Topic string // Topic name. Will create it if not exists.
}

type AzureBlobStorageOptions struct {
	Container string // Container name. Will create it if it does not exists.
	Blob      string // Blob name to use, will be created inside the container.
	Port      string // Need port number for tests, to update the connection string
}

type LumberjackOptions struct {
	ParseJSON bool // Parse the input bytes as JSON and send structured data. By default, input bytes are sent in a 'message' field.
}

type GcsOptions struct {
	ProjectID string // Project ID, needs to be unique with multiple buckets of the same name.
	Bucket    string // Bucket name. Will create it if do not exist.
	Object    string // Name of the object created inside the related Bucket.
}
