package cmd

import (
	"fmt"

	"github.com/k0sproject/k0sctl/action"
	"github.com/k0sproject/k0sctl/phase"

	"github.com/urfave/cli/v2"
)

var statusCommand = &cli.Command{
	Name:  "status",
	Usage: "Show the current status of the k0s cluster",
	Flags: []cli.Flag{
		configFlag,
		concurrencyFlag,
		&cli.StringFlag{
			Name:    "output",
			Usage:   "Output format (default: human-readable, json)",
			Aliases: []string{"o"},
		},
		debugFlag,
		traceFlag,
		redactFlag,
		timeoutFlag,
	},
	Before: actions(initLogging, initConfig, initManager, displayCopyright),
	After:  actions(cancelTimeout),
	Action: func(ctx *cli.Context) error {
		manager, ok := ctx.Context.Value(ctxManagerKey{}).(*phase.Manager)
		if !ok {
			return fmt.Errorf("failed to retrieve manager from context")
		}

		statusAction := action.NewStatus(action.StatusOptions{
			Manager: manager,
			Writer:  ctx.App.Writer,
			Format:  ctx.String("output"),
		})

		if err := statusAction.Run(ctx.Context); err != nil {
			return fmt.Errorf("status check failed - log file saved to %s: %w", ctx.Context.Value(ctxLogFileKey{}).(string), err)
		}

		return nil
	},
}
