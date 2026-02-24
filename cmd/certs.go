package cmd

import (
	"github.com/k0sproject/k0sctl/action"
	"github.com/k0sproject/k0sctl/phase"

	"github.com/urfave/cli/v2"
)

var certsCommand = &cli.Command{
	Name:  "certs",
	Usage: "Certificate management commands",
	Subcommands: []*cli.Command{
		certsCheckCommand,
		certsRenewCommand,
	},
}

var certsCheckCommand = &cli.Command{
	Name:  "check",
	Usage: "Check certificate expiry status on all controllers",
	Flags: []cli.Flag{
		configFlag,
		debugFlag,
		traceFlag,
		redactFlag,
		timeoutFlag,
		concurrencyFlag,
		&cli.IntFlag{
			Name:    "expiry-warning-days",
			Usage:   "Warn about certificates expiring within this many days",
			Aliases: []string{"days"},
			Value:   30,
		},
	},
	Before: actions(initLogging, displayCopyright, initManager),
	After:  actions(cancelTimeout),
	Action: func(ctx *cli.Context) error {
		certsCheck := action.NewCertsCheck(action.CertsCheckOptions{
			Manager:           ctx.Context.Value(ctxManagerKey{}).(*phase.Manager),
			ExpiryWarningDays: ctx.Int("expiry-warning-days"),
		})

		return certsCheck.Run(ctx.Context)
	},
}

var certsRenewCommand = &cli.Command{
	Name:  "renew",
	Usage: "Renew leaf certificates by performing a rolling restart of controllers",
	Description: `Renews non-CA certificates by restarting k0s controllers one-by-one.
k0s automatically rotates leaf certificates (apiserver, kubelet, etcd peer/client,
etc.) on startup.

CA certificates are NOT renewed by this command. CA replacement requires a manual
process involving stopping all nodes, regenerating the CA, and redistributing it.
See: https://docs.k0sproject.io/stable/troubleshooting/certificate-authorities/`,
	Flags: []cli.Flag{
		configFlag,
		debugFlag,
		traceFlag,
		redactFlag,
		timeoutFlag,
		concurrencyFlag,
		dryRunFlag,
	},
	Before: actions(initLogging, displayCopyright, initManager),
	After:  actions(cancelTimeout),
	Action: func(ctx *cli.Context) error {
		certsRenew := action.NewCertsRenew(action.CertsRenewOptions{
			Manager: ctx.Context.Value(ctxManagerKey{}).(*phase.Manager),
		})

		return certsRenew.Run(ctx.Context)
	},
}
