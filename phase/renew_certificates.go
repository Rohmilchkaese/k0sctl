package phase

import (
	"context"
	"fmt"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/k0sctl/pkg/node"
	"github.com/k0sproject/k0sctl/pkg/retry"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// RenewCertificates renews k0s certificates by performing a rolling restart
// of controllers. k0s automatically rotates non-CA certificates on startup,
// so restarting the service is sufficient to get fresh certificates.
//
// CA certificates (10-year default lifetime) are NOT renewed by this phase.
// CA replacement requires a manual process involving stopping all nodes,
// regenerating the CA on one controller, and redistributing it.
type RenewCertificates struct {
	GenericPhase

	controllers cluster.Hosts
	leader      *cluster.Host
}

// Title for the phase
func (p *RenewCertificates) Title() string {
	return "Renew certificates"
}

// Prepare the phase
func (p *RenewCertificates) Prepare(config *v1beta1.Cluster) error {
	p.Config = config
	p.leader = p.Config.Spec.K0sLeader()
	p.controllers = p.Config.Spec.Hosts.Controllers().Filter(func(h *cluster.Host) bool {
		return h.Metadata.K0sRunningVersion != nil && h.IsConnected()
	})
	return nil
}

// ShouldRun when there are running controllers.
func (p *RenewCertificates) ShouldRun() bool {
	return len(p.controllers) > 0
}

// Run the phase. Controllers are restarted one-by-one to maintain availability.
func (p *RenewCertificates) Run(ctx context.Context) error {
	log.Infof("performing rolling restart of %d controller(s) to renew certificates", len(p.controllers))
	log.Warnf("note: CA certificates are NOT renewed by this operation (they have a 10-year lifetime)")
	log.Warnf("only leaf certificates (apiserver, kubelet, etcd peer/client, etc.) are rotated on restart")

	for _, h := range p.controllers {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := p.restartController(ctx, h); err != nil {
			return fmt.Errorf("%s: failed to renew certificates: %w", h, err)
		}
	}

	log.Infof("certificate renewal complete: all %d controller(s) restarted successfully", len(p.controllers))
	return nil
}

func (p *RenewCertificates) restartController(ctx context.Context, h *cluster.Host) error {
	serviceName := h.K0sServiceName()

	log.Infof("%s: stopping k0s service for certificate renewal", h)
	err := p.Wet(h, "stop k0s service for certificate renewal", func() error {
		if err := h.Configurer.StopService(h, serviceName); err != nil {
			return err
		}
		if err := retry.WithDefaultTimeout(ctx, node.ServiceStoppedFunc(h, serviceName)); err != nil {
			return fmt.Errorf("wait for k0s service stop: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Infof("%s: starting k0s service (certificates will be rotated on startup)", h)
	err = p.Wet(h, "start k0s service to rotate certificates", func() error {
		if err := h.Configurer.StartService(h, serviceName); err != nil {
			return err
		}
		if err := retry.WithDefaultTimeout(ctx, node.ServiceRunningFunc(h, serviceName)); err != nil {
			return fmt.Errorf("wait for k0s service start: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Wait for the API server to become ready after restart.
	if p.IsWet() {
		log.Infof("%s: waiting for API server readiness", h)
		err := retry.WithDefaultTimeout(ctx, func(_ context.Context) error {
			out, err := h.ExecOutput(
				h.Configurer.KubectlCmdf(h, h.K0sDataDir(), "get --raw='/readyz'"),
				exec.Sudo(h),
			)
			if err != nil {
				return fmt.Errorf("readiness endpoint: %q: %w", out, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("controller did not reach ready state after restart: %w", err)
		}
	}

	log.Infof("%s: certificate renewal complete", h)
	return nil
}
