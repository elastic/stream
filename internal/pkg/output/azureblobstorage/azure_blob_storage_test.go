// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package azureblobstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/elastic/stream/internal/pkg/output"
)

const (
	emulatorHost = "127.0.0.1"
	emulatorPort = "10000"
	container    = "testcontainer"
	blob         = "testblob"
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mcr.microsoft.com/azure-storage/azurite",
		Tag:        "latest",
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
		resp, err := httpClient.Get(fmt.Sprintf("http://%s:%s/devstoreaccount1", emulatorHost, emulatorPort))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
		}

		return nil
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not connect to the Azure Blob Storage instance: %s", err)
	}

	code := m.Run()

	_ = pool.Purge(resource)

	os.Exit(code)
}

func TestAzureBlobStorage(t *testing.T) {
	out, err := New(&output.Options{
		Addr: emulatorHost,
		AzureBlobStorageOptions: output.AzureBlobStorageOptions{
			Container: container,
			Blob:      blob,
			Port:      emulatorPort,
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

	connectionString := fmt.Sprintf("DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://%s:%s/devstoreaccount1;", emulatorHost, emulatorPort)
	ctx, cancel := context.WithCancel(context.Background())
	serviceClient, _ := azblob.NewClientFromConnectionString(connectionString, nil)

	blobDownloadResponse, err := serviceClient.DownloadStream(ctx, container, blob, nil)
	require.NoError(t, err)

	reader := blobDownloadResponse.Body
	downloadData, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(downloadData))

	err = reader.Close()
	require.NoError(t, err)

	t.Cleanup(cancel)
}
