// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package gcppubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andrewkroh/stream/pkg/output"
)

const (
	emulatorHost = "127.0.0.1"
	emulatorPort = "8681"
	project      = "testProject"
	topic        = "testTopic"
	subscription = "testSubscription"
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "google/cloud-sdk",
		Tag:        "emulators",
		Cmd:        []string{"gcloud", "beta", "emulators", "pubsub", "start", fmt.Sprintf("--host-port=0.0.0.0:%s", emulatorPort)},
		PortBindings: map[docker.Port][]docker.PortBinding{
			emulatorPort: {{HostIP: emulatorHost, HostPort: emulatorPort}},
		},
		ExposedPorts: []string{emulatorPort},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry
	if err := pool.Retry(func() error {
		// Disable HTTP keep-alives to ensure no extra goroutines hang around.
		httpClient := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

		// Sanity check the emulator.
		resp, err := httpClient.Get(fmt.Sprintf("http://%s:%s", emulatorHost, emulatorPort))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
		}

		return nil
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not connect to the gcp pubsub emulator: %s", err)
	}

	code := m.Run()

	_ = pool.Purge(resource)

	os.Exit(code)
}

func TestGCPPubsub(t *testing.T) {
	out, err := New(&output.Options{
		Addr: fmt.Sprintf("%s:%s", emulatorHost, emulatorPort),
		GCPPubsubOptions: output.GCPPubsubOptions{
			Project:      project,
			Topic:        topic,
			Subscription: subscription,
			Clear:        true,
		},
	})
	require.NoError(t, err)

	err = out.DialContext(context.Background())
	require.NoError(t, err)

	event := map[string]interface{}{
		"message": "hello world!",
	}
	data, err := json.Marshal(event)
	require.NoError(t, err)

	n, err := out.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	ctx, cancel := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, project)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(cancel)

	recvCtx, recvCancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(recvCancel)
	var recvData []byte
	require.NoError(t, client.Subscription(subscription).Receive(recvCtx, func(_ context.Context, msg *pubsub.Message) {
		recvData = msg.Data
		msg.Ack()
		recvCancel()
	}))
	assert.Equal(t, string(data), string(recvData))
}
