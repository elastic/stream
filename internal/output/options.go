// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package output

import "time"

// Options holds the configuration for an output.
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
	AzureEventHubOptions
	LumberjackOptions
	GCSOptions
}

// WebhookOptions holds configuration for the webhook output.
type WebhookOptions struct {
	ContentType string        // Content-Type header.
	Headers     []string      // Headers in Key=Value format.
	Username    string        // Basic auth username.
	Password    string        // Basic auth password.
	Timeout     time.Duration // Timeout for request handling.
	Probe       string        // Server probe behavior.
}

// GCPPubsubOptions holds configuration for the Google Cloud Pub/Sub output.
type GCPPubsubOptions struct {
	Project      string // Project name.
	Topic        string // Topic name. Will create it if not exists.
	Subscription string // Subscription name. Will create it if not exists.
	Clear        bool   // Clear will clear all topics and subscriptions before running.
}

// KafkaOptions holds configuration for the Kafka output.
type KafkaOptions struct {
	Topic string // Topic is the Kafka topic name. It will be created if it does not exist.
}

// AzureBlobStorageOptions holds configuration for the Azure Blob Storage output.
type AzureBlobStorageOptions struct {
	Container string // Container is the container name. It will be created if it does not exist.
	Blob      string // Blob is the blob name to use. It will be created inside the container.
	Port      string // Port is the port number used for tests to update the connection string.
}

// AzureEventHubOptions holds configuration for the Azure Event Hub output.
type AzureEventHubOptions struct {
	FullyQualifiedNamespace string // FullyQualifiedNamespace is the Event Hubs namespace name (e.g. myeventhub.servicebus.windows.net).
	EventHubName            string // EventHubName is the name of the Event Hub.
	ConnectionString        string // ConnectionString is the connection string to connect to the Event Hub.
}

// LumberjackOptions holds configuration for the Lumberjack output.
type LumberjackOptions struct {
	// ParseJSON parses the input bytes as JSON and sends structured data. By default, input bytes are sent in a 'message' field.
	ParseJSON bool
}

// GCSOptions holds configuration for the Google Cloud Storage output.
type GCSOptions struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string
	// ObjectContentType is the content-type set for the object that is created in the bucket. Defaults to application/json.
	ObjectContentType string
	// Bucket is the bucket name. It will be created if it does not exist.
	Bucket string
	// Object is the name of the object created inside the related bucket.
	Object string
}
