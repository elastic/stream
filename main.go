// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"os"

	"github.com/elastic/stream/internal/command"
)

func main() {
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
