package action

import (
	"context"

	"github.com/k0sproject/k0sctl/phase"

	log "github.com/sirupsen/logrus"
)

// CertsCheckOptions configures the certificate check action.
type CertsCheckOptions struct {
	Manager           *phase.Manager
	ExpiryWarningDays int
}

type CertsCheck struct {
	CertsCheckOptions
}

func NewCertsCheck(opts CertsCheckOptions) *CertsCheck {
	checkPhase := &phase.CheckCertificates{
		ExpiryWarningDays: opts.ExpiryWarningDays,
	}

	c := &CertsCheck{CertsCheckOptions: opts}

	c.Manager.SetPhases(phase.Phases{
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&phase.GatherK0sFacts{},
		checkPhase,
		&phase.Disconnect{},
	})

	return c
}

func (c *CertsCheck) Run(ctx context.Context) error {
	if err := c.Manager.Run(ctx); err != nil {
		log.Info(phase.Colorize.Red("==> Certificate check failed").String())
		return err
	}
	return nil
}

// CertsRenewOptions configures the certificate renewal action.
type CertsRenewOptions struct {
	Manager *phase.Manager
}

type CertsRenew struct {
	CertsRenewOptions
}

func NewCertsRenew(opts CertsRenewOptions) *CertsRenew {
	c := &CertsRenew{CertsRenewOptions: opts}

	c.Manager.SetPhases(phase.Phases{
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&phase.GatherK0sFacts{},
		&phase.RenewCertificates{},
		&phase.Disconnect{},
	})

	return c
}

func (c *CertsRenew) Run(ctx context.Context) error {
	if err := c.Manager.Run(ctx); err != nil {
		log.Info(phase.Colorize.Red("==> Certificate renewal failed").String())
		return err
	}
	log.Info(phase.Colorize.Green("==> Certificate renewal complete").String())
	return nil
}
