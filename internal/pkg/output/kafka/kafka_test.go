// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package kafka

import (
	"log"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/elastic/stream/internal/pkg/output"
)

const (
	emulatorHost = "127.0.0.1"
	emulatorPort = "9092"
	topic        = "testTopic"
)

var (
	outputEvent         string
	emulatorHostAndPort = net.JoinHostPort(emulatorHost, emulatorPort)
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "bashj79/kafka-kraft",
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
		config := sarama.NewConfig()
		config.Producer.Partitioner = sarama.NewRandomPartitioner
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Producer.Return.Successes = true
		_, err := sarama.NewClient([]string{emulatorHostAndPort}, config)
		return err
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not connect to the kafka instance: %s", err)
	}

	code := m.Run()

	_ = pool.Purge(resource)

	os.Exit(code)
}

func TestKafka(t *testing.T) {
	out, err := New(&output.Options{
		Addr: emulatorHostAndPort,
		KafkaOptions: output.KafkaOptions{
			Topic: topic,
		},
	})
	require.NoError(t, err)

	event := "testmessage something"

	n, err := out.Write([]byte(event))
	require.NoError(t, err)
	assert.Equal(t, len(event), n)

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	consumer, err := sarama.NewConsumer(strings.Split(emulatorHostAndPort, ","), config)
	require.NoError(t, err)

	defer consumer.Close()

	pc, err := consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		t.Fatalf("Failed to start partition consumer: %v", err)
		return
	}

	for {
		select {
		case err := <-pc.Errors():
			t.Fatal(err)
		case msg := <-pc.Messages():
			outputEvent = string(msg.Value)
			assert.Equal(t, event, outputEvent)
			return
		}
	}
}
