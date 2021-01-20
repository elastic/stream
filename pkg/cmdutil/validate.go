package cmdutil

import (
	"fmt"
	"os"

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
	for _, f := range args {
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
