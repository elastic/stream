// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "UNKNOWN"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output version",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("stream %s\n", version)
	},
}
