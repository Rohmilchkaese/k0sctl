package phase

import (
	"context"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	log "github.com/sirupsen/logrus"
)

// DetectReplacedNodes identifies hosts that appear to have been replaced
// (reachable but without k0s installed) and logs informational messages.
type DetectReplacedNodes struct {
	GenericPhase
	replaced cluster.Hosts
}

// Title for the phase
func (p *DetectReplacedNodes) Title() string {
	return "Detect replaced nodes"
}

// Prepare the phase
func (p *DetectReplacedNodes) Prepare(config *v1beta1.Cluster) error {
	p.Config = config
	p.replaced = config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		// A "replaced" node is one that:
		// 1. Is not explicitly marked for reset
		// 2. Has no k0s binary (not installed)
		// 3. Has no running k0s version
		// These conditions mean the host is reachable (we connected in the Connect phase)
		// but has no k0s on it — likely a fresh OS on the same IP.
		if h.Reset {
			return false
		}
		return h.Metadata.K0sBinaryVersion == nil && h.Metadata.K0sRunningVersion == nil
	})
	return nil
}

// ShouldRun is true when there are hosts that look replaced
func (p *DetectReplacedNodes) ShouldRun() bool {
	return len(p.replaced) > 0
}

// Run the phase
func (p *DetectReplacedNodes) Run(_ context.Context) error {
	for _, h := range p.replaced {
		log.Infof("%s: host appears to be new or replaced (reachable but k0s is not installed), will be provisioned as a fresh %s node", h, h.Role)

		if h.IsController() {
			log.Warnf("%s: if this controller was previously part of the cluster, ensure the old etcd member has been removed (use --force to auto-remove, or run `k0s etcd leave --peer-address %s` on another controller)", h, h.PrivateAddress)
		}
	}

	if len(p.replaced) > 0 {
		log.Infof("%d node(s) detected as new/replaced and will be provisioned", len(p.replaced))
	}

	return nil
}
