package action

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/k0sproject/k0sctl/phase"

	log "github.com/sirupsen/logrus"
)

type StatusOptions struct {
	// Manager is the phase manager
	Manager *phase.Manager
	// Writer is where status output goes
	Writer io.Writer
	// Format controls output format: "" for human-readable, "json" for JSON
	Format string
}

type Status struct {
	StatusOptions
}

func NewStatus(opts StatusOptions) *Status {
	return &Status{StatusOptions: opts}
}

func (s *Status) Run(ctx context.Context) error {
	start := time.Now()

	phases := phase.Phases{
		&phase.DefaultK0sVersion{},
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{SkipMachineIDs: true},
		&phase.GatherK0sFacts{},
		&phase.ReportStatus{
			Writer: s.Writer,
			Format: s.Format,
		},
		&phase.Disconnect{},
	}

	s.Manager.SetPhases(phases)

	if err := s.Manager.Run(ctx); err != nil {
		log.Infof(phase.Colorize.Red("==> Status check failed").String())
		return err
	}

	duration := time.Since(start).Truncate(time.Second)
	log.Infof(phase.Colorize.Green(fmt.Sprintf("==> Status collected in %s", duration)).String())

	return nil
}
