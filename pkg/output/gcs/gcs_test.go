// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/elastic/stream/pkg/output"
)

const (
	emulatorHost = "localhost"
	emulatorPort = "4443"
	bucket       = "testbucket"
	objectname   = "testobject"
)

var emulatorHostAndPort = fmt.Sprintf("http://%s", net.JoinHostPort(emulatorHost, emulatorPort))

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "fsouza/fake-gcs-server",
		Tag:        "latest",
		Cmd:        []string{"-host=0.0.0.0", "-public-host=localhost", fmt.Sprintf("-port=%s", emulatorPort), "-scheme=http"},
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

	if err := pool.Retry(func() error {
		// Disable HTTP keep-alives to ensure no extra goroutines hang around.
		httpClient := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

		// Sanity check the emulator.
		resp, err := httpClient.Get(fmt.Sprintf("http://%s:%s/storage/v1/b", emulatorHost, emulatorPort))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(resp.StatusCode)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
		}
		return err
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not connect to the gcs instance: %s", err)
	}

	code := m.Run()

	_ = pool.Purge(resource)

	os.Exit(code)
}

func TestGcs(t *testing.T) {
	out, err := New(&output.Options{
		Addr: emulatorHostAndPort,
		GcsOptions: output.GcsOptions{
			Bucket: bucket,
			Object: objectname,
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

	// Need to close the Writer for the Object to be created in the Bucket.
	out.Close()

	os.Setenv("STORAGE_EMULATOR_HOST", emulatorHostAndPort)

	ctx, cancel := context.WithCancel(context.Background())
	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(cancel)

	o := client.Bucket(bucket).Object(objectname)
	r, err := o.NewReader(ctx)
	require.NoError(t, err)

	body, err := io.ReadAll(r)
	require.NoError(t, err)
	defer r.Close()

	assert.Equal(t, string(data), string(body))
}
