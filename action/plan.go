package action

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/k0sproject/k0sctl/phase"

	log "github.com/sirupsen/logrus"
)

type PlanOptions struct {
	// Manager is the phase manager
	Manager *phase.Manager
	// Writer is where plan output goes
	Writer io.Writer
	// DisableDowngradeCheck skips the downgrade check
	DisableDowngradeCheck bool
}

type Plan struct {
	PlanOptions
}

func NewPlan(opts PlanOptions) *Plan {
	return &Plan{PlanOptions: opts}
}

func (p *Plan) Run(ctx context.Context) error {
	start := time.Now()

	// Force dry-run mode so no cluster state is altered.
	p.Manager.DryRun = true

	apply := NewApply(ApplyOptions{
		Manager:               p.Manager,
		NoWait:                true,
		NoDrain:               true,
		DisableDowngradeCheck: p.DisableDowngradeCheck,
	})

	p.Manager.SetPhases(apply.Phases)

	if err := p.Manager.Run(ctx); err != nil {
		log.Info(phase.Colorize.Red("==> Plan failed").String())
		return err
	}

	duration := time.Since(start).Truncate(time.Second)
	log.Info(phase.Colorize.Green(fmt.Sprintf("==> Plan completed in %s", duration)).String())

	return nil
}
