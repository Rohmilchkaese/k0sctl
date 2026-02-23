package phase

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	log "github.com/sirupsen/logrus"
)

// ResetKnownHosts removes SSH known_hosts entries for reset hosts.
// This prevents "host key mismatch" errors when a new machine
// gets the same IP as a previously reset node.
type ResetKnownHosts struct {
	GenericPhase
}

// Title for the phase
func (p *ResetKnownHosts) Title() string {
	return "Remove known_hosts entries for reset hosts"
}

// Run the phase
func (p *ResetKnownHosts) Run(_ context.Context) error {
	for _, h := range p.Config.Spec.Hosts {
		if !h.Reset {
			continue
		}
		if h.Protocol() != "ssh" {
			continue
		}
		p.removeKnownHost(h)
	}
	return nil
}

func (p *ResetKnownHosts) removeKnownHost(h *cluster.Host) {
	addr := h.Address()
	if addr == "" || addr == "127.0.0.1" {
		return
	}

	log.Infof("%s: removing known_hosts entry for %s", h, addr)
	out, err := exec.Command("ssh-keygen", "-R", addr).CombinedOutput()
	if err != nil {
		log.Debugf("%s: ssh-keygen -R %s: %s", h, addr, out)
		log.Warnf("%s: failed to remove known_hosts entry: %v", h, err)
		return
	}

	// If the host uses a non-standard port, also remove the [host]:port form
	if h.SSH != nil && h.SSH.Port != 0 && h.SSH.Port != 22 {
		bracketed := fmt.Sprintf("[%s]:%d", addr, h.SSH.Port)
		log.Debugf("%s: also removing known_hosts entry for %s", h, bracketed)
		out, err := exec.Command("ssh-keygen", "-R", bracketed).CombinedOutput()
		if err != nil {
			log.Debugf("%s: ssh-keygen -R %s: %s", h, bracketed, out)
			log.Warnf("%s: failed to remove known_hosts entry for %s: %v", h, bracketed, err)
		}
	}
}
