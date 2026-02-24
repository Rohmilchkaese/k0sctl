package cmd

import (
	"fmt"

	"github.com/k0sproject/k0sctl/action"
	"github.com/k0sproject/k0sctl/phase"

	"github.com/urfave/cli/v2"
)

var planCommand = &cli.Command{
	Name:  "plan",
	Usage: "Show what changes would be made to the cluster (dry-run)",
	Flags: []cli.Flag{
		configFlag,
		concurrencyFlag,
		concurrentUploadsFlag,
		&cli.BoolFlag{
			Name:   "disable-downgrade-check",
			Usage:  "Skip downgrade check",
			Hidden: true,
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

		planAction := action.NewPlan(action.PlanOptions{
			Manager:               manager,
			Writer:                ctx.App.Writer,
			DisableDowngradeCheck: ctx.Bool("disable-downgrade-check"),
		})

		if err := planAction.Run(ctx.Context); err != nil {
			return fmt.Errorf("plan failed - log file saved to %s: %w", ctx.Context.Value(ctxLogFileKey{}).(string), err)
		}

		return nil
	},
}
