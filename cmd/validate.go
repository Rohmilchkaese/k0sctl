package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// validateCommand provides a way to check a k0sctl configuration file for
// errors without connecting to any hosts or performing any cluster operations.
// This is useful for CI pipelines and for catching configuration mistakes
// before running `k0sctl apply`.
var validateCommand = &cli.Command{
	Name:  "validate",
	Usage: "Validate a k0sctl configuration file",
	Flags: []cli.Flag{
		configFlag,
	},
	Before: actions(initLogging, initConfig, displayCopyright),
	Action: func(ctx *cli.Context) error {
		cfg, err := readConfig(ctx)
		if err != nil {
			return err
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Fprintln(ctx.App.Writer, "configuration is valid")
		return nil
	},
}
