package phase

import (
	"context"
	"fmt"
	"slices"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	log "github.com/sirupsen/logrus"
)

// ReconcileEtcdMembers validates and reconciles etcd cluster membership.
//
// This phase extends the original ValidateEtcdMembers with automatic cleanup
// of orphaned etcd members — members whose IP no longer matches any controller
// in the k0sctl configuration. This addresses the common day-2 scenario where
// a controller node dies and is replaced (possibly with a different IP), leaving
// a dead member in the etcd cluster that degrades quorum tolerance.
//
// Automatic removal requires --force and is guarded by quorum safety checks:
// a member is only removed if the remaining healthy members still form a majority.
type ReconcileEtcdMembers struct {
	GenericPhase

	// newControllers are controllers listed in YAML that don't have k0s running.
	newControllers cluster.Hosts
	// orphanedMembers are etcd member IPs that don't match any controller in YAML.
	orphanedMembers []string
	leader          *cluster.Host
}

// Title for the phase
func (p *ReconcileEtcdMembers) Title() string {
	return "Reconcile etcd members"
}

// Prepare the phase
func (p *ReconcileEtcdMembers) Prepare(config *v1beta1.Cluster) error {
	p.Config = config
	p.leader = p.Config.Spec.K0sLeader()

	// Find new controllers (no k0s running yet).
	p.newControllers = p.Config.Spec.Hosts.Controllers().Filter(func(h *cluster.Host) bool {
		return h.Metadata.K0sRunningVersion == nil
	})

	// Find orphaned etcd members: IPs in the etcd member list that don't match
	// any controller's private address (or public address) in the YAML.
	p.orphanedMembers = p.findOrphanedMembers()

	return nil
}

// ShouldRun is true when there's an existing cluster with etcd to check.
func (p *ReconcileEtcdMembers) ShouldRun() bool {
	if p.leader == nil {
		return false
	}
	if p.leader.Metadata.K0sRunningVersion == nil {
		log.Debugf("%s: leader has no k0s running, assuming a fresh cluster", p.leader)
		return false
	}
	if p.leader.Role == "single" {
		log.Debugf("%s: leader is a single node, assuming no etcd", p.leader)
		return false
	}
	if s := p.Config.StorageType(); s != "etcd" {
		log.Debugf("%s: storage type is %q, not k0s managed etcd", p.leader, s)
		return false
	}

	return len(p.newControllers) > 0 || len(p.orphanedMembers) > 0
}

// Run the phase
func (p *ReconcileEtcdMembers) Run(_ context.Context) error {
	// First handle orphaned members (members not in YAML at all).
	if err := p.reconcileOrphanedMembers(); err != nil {
		return err
	}

	// Then handle the controller-swap case (new controller with IP already in etcd).
	if err := p.validateControllerSwap(); err != nil {
		return err
	}

	return nil
}

// findOrphanedMembers returns etcd member IPs that don't correspond to any
// controller in the k0sctl configuration.
func (p *ReconcileEtcdMembers) findOrphanedMembers() []string {
	if len(p.Config.Metadata.EtcdMembers) == 0 {
		return nil
	}

	// Build a set of all known controller addresses (private + public).
	knownAddrs := make(map[string]bool)
	for _, h := range p.Config.Spec.Hosts.Controllers() {
		if h.PrivateAddress != "" {
			knownAddrs[h.PrivateAddress] = true
		}
		knownAddrs[h.Address()] = true
	}

	var orphaned []string
	for _, memberIP := range p.Config.Metadata.EtcdMembers {
		if !knownAddrs[memberIP] {
			orphaned = append(orphaned, memberIP)
		}
	}

	return orphaned
}

// reconcileOrphanedMembers handles etcd members whose IPs don't match any
// controller in the configuration. With --force, it removes them after
// verifying quorum safety. Without --force, it warns the user.
func (p *ReconcileEtcdMembers) reconcileOrphanedMembers() error {
	if len(p.orphanedMembers) == 0 {
		return nil
	}

	totalMembers := len(p.Config.Metadata.EtcdMembers)
	numOrphaned := len(p.orphanedMembers)
	remaining := totalMembers - numOrphaned
	quorum := (totalMembers / 2) + 1

	log.Warnf("found %d etcd member(s) not matching any controller in the configuration: %v", numOrphaned, p.orphanedMembers)
	log.Warnf("etcd cluster has %d members total, %d in config, quorum requires %d", totalMembers, remaining, quorum)

	if !Force {
		log.Warnf("orphaned etcd members will NOT be removed without --force")
		log.Warnf("to remove them manually, run on a healthy controller:")
		for _, ip := range p.orphanedMembers {
			log.Warnf("  k0s etcd leave --peer-address %s", ip)
		}
		log.Warnf("or re-run apply with --force to remove them automatically")
		return nil
	}

	// With --force: check if removal is quorum-safe.
	if remaining < quorum {
		return fmt.Errorf(
			"cannot safely remove %d orphaned etcd member(s): "+
				"cluster has %d members, removing %d would leave %d which is below quorum (%d). "+
				"this likely means the cluster has already lost quorum and requires manual etcd disaster recovery",
			numOrphaned, totalMembers, numOrphaned, remaining, quorum,
		)
	}

	// Also verify the remaining members are actually healthy (they have
	// k0s running and are connected). Count how many of the non-orphaned
	// members correspond to a running, connected controller.
	healthyRemaining := 0
	for _, h := range p.Config.Spec.Hosts.Controllers() {
		if h.Metadata.K0sRunningVersion != nil && h.IsConnected() {
			healthyRemaining++
		}
	}

	healthyQuorum := (totalMembers / 2) + 1
	if healthyRemaining < healthyQuorum {
		return fmt.Errorf(
			"cannot safely remove orphaned etcd members: "+
				"only %d of %d remaining controllers are healthy and connected, "+
				"which is below quorum (%d). manual etcd recovery may be required",
			healthyRemaining, remaining, healthyQuorum,
		)
	}

	log.Infof("quorum check passed: %d healthy controllers remain (quorum=%d), proceeding with removal", healthyRemaining, healthyQuorum)

	// Remove each orphaned member.
	for _, ip := range p.orphanedMembers {
		log.Infof("removing orphaned etcd member with peer address %s", ip)

		leaveCmd := p.leader.Configurer.K0sCmdf("etcd leave --peer-address %s --data-dir=%s", ip, p.leader.K0sDataDir())

		err := p.Wet(nil, fmt.Sprintf("remove orphaned etcd member %s", ip), func() error {
			return p.leader.Exec(leaveCmd)
		})
		if err != nil {
			return fmt.Errorf("failed to remove orphaned etcd member %s: %w. "+
				"you may need to run manually on a controller: k0s etcd leave --peer-address %s",
				ip, err, ip)
		}

		log.Infof("successfully removed orphaned etcd member %s", ip)
	}

	return nil
}

// validateControllerSwap handles the case where a new controller's IP is
// already in the etcd member list (i.e., a node was replaced with the same IP).
// This preserves the original ValidateEtcdMembers behavior.
func (p *ReconcileEtcdMembers) validateControllerSwap() error {
	if len(p.Config.Metadata.EtcdMembers) > len(p.Config.Spec.Hosts.Controllers()) {
		log.Warnf("there are more etcd members in the cluster than controllers listed in the configuration")
	}

	for _, h := range p.newControllers {
		log.Debugf("%s: host is new, checking if etcd members list already contains %s", h, h.PrivateAddress)
		if slices.Contains(p.Config.Metadata.EtcdMembers, h.PrivateAddress) {
			if Force {
				log.Infof("%s: force used, running 'k0s etcd leave' for the host", h)
				leaveCommand := p.leader.Configurer.K0sCmdf("etcd leave --peer-address %s --data-dir=%s", h.PrivateAddress, p.leader.K0sDataDir())
				err := p.Wet(h, fmt.Sprintf("remove host from etcd using %v", leaveCommand), func() error {
					return p.leader.Exec(leaveCommand)
				})
				if err != nil {
					return fmt.Errorf("controller %s is listed as an existing etcd member but k0s is not found installed on it, "+
						"the host may have been replaced. attempted etcd leave for the address %s but it failed: %w", h, h.PrivateAddress, err)
				}
				continue
			}
			return fmt.Errorf("controller %s is listed as an existing etcd member but k0s is not found installed on it, "+
				"the host may have been replaced. check the host and use `k0s etcd leave --peer-address %s` "+
				"on a controller or re-run apply with --force", h, h.PrivateAddress)
		}
		log.Debugf("%s: no match, assuming it's safe to install", h)
	}

	return nil
}
