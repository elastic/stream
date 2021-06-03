// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// ValidateArgs combines PositionalArgs validators and runs them serially.
func ValidateArgs(validators ...cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range validators {
			if err := f(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// RegularFiles validates that each arg is a regular file.
func RegularFiles(_ *cobra.Command, args []string) error {
	paths, err := ExpandGlobPatternsFromArgs(args)
	if err != nil {
		return err
	}
	for _, f := range paths {
		info, err := os.Stat(f)
		if err != nil {
			return fmt.Errorf("arg %q is not a valid file: %w", f, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("arg %q is not a regular file", f)
		}
	}
	return nil
}

func ExpandGlobPatternsFromArgs(args []string) ([]string, error) {
	var paths []string
	for _, pat := range args {
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pat, err)
		}
		paths = append(paths, matches...)
	}
	return paths, nil
}
