package phase

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

const (
	// Minimum recommended free disk space in bytes (2 GiB).
	minDiskSpaceBytes = 2 * 1024 * 1024 * 1024
	// Maximum acceptable clock skew in seconds between nodes.
	maxClockSkewSeconds = 5
)

// PreflightChecks runs pre-apply validation on every host.
type PreflightChecks struct {
	GenericPhase
}

// Title for the phase
func (p *PreflightChecks) Title() string {
	return "Run preflight checks"
}

// Run the phase
func (p *PreflightChecks) Run(ctx context.Context) error {
	var warnings []string

	// Check each host in parallel for disk space and basic connectivity.
	if err := p.parallelDo(ctx, p.Config.Spec.Hosts, func(_ context.Context, h *cluster.Host) error {
		if err := p.checkDiskSpace(h); err != nil {
			log.Warnf("%s: %v", h, err)
			warnings = append(warnings, fmt.Sprintf("%s: %v", h, err))
		}
		return nil
	}); err != nil {
		return err
	}

	// Check clock skew between nodes.
	if err := p.checkClockSkew(); err != nil {
		log.Warnf("clock skew check: %v", err)
		warnings = append(warnings, fmt.Sprintf("clock skew: %v", err))
	}

	// Check controller-to-controller connectivity (etcd peering ports).
	controllers := p.Config.Spec.Hosts.Controllers()
	if len(controllers) > 1 {
		if err := p.checkControllerConnectivity(controllers); err != nil {
			log.Warnf("controller connectivity: %v", err)
			warnings = append(warnings, fmt.Sprintf("controller connectivity: %v", err))
		}
	}

	if len(warnings) > 0 {
		log.Warnf("preflight checks completed with %d warning(s)", len(warnings))
	} else {
		log.Infof("all preflight checks passed")
	}

	return nil
}

// checkDiskSpace verifies the host has at least minDiskSpaceBytes of free space
// on the filesystem containing the data directory.
func (p *PreflightChecks) checkDiskSpace(h *cluster.Host) error {
	dataDir := h.K0sDataDir()

	// Use df to check available space on the partition containing the data dir.
	// Some systems may not have the dir yet; fall back to /var/lib.
	checkDir := dataDir
	output, err := h.ExecOutput(fmt.Sprintf("df -B1 --output=avail %s 2>/dev/null || df -B1 --output=avail /var/lib 2>/dev/null || echo unknown", checkDir), exec.Sudo(h))
	if err != nil || strings.Contains(output, "unknown") {
		log.Debugf("%s: could not check disk space: %v", h, err)
		return nil // skip if df is not available
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil
	}

	avail, err := strconv.ParseInt(strings.TrimSpace(lines[len(lines)-1]), 10, 64)
	if err != nil {
		log.Debugf("%s: could not parse df output: %v", h, err)
		return nil
	}

	if avail < minDiskSpaceBytes {
		return fmt.Errorf("low disk space: %s available, recommend at least %s",
			humanBytes(avail), humanBytes(minDiskSpaceBytes))
	}

	log.Debugf("%s: disk space OK (%s available)", h, humanBytes(avail))
	return nil
}

// checkClockSkew compares the system time across all hosts and reports if the
// maximum skew exceeds the threshold.
func (p *PreflightChecks) checkClockSkew() error {
	type hostTime struct {
		host string
		ts   int64
	}

	var times []hostTime

	for _, h := range p.Config.Spec.Hosts {
		output, err := h.ExecOutput("date +%s", exec.Sudo(h))
		if err != nil {
			log.Debugf("%s: could not get time: %v", h, err)
			continue
		}
		ts, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
		if err != nil {
			continue
		}
		times = append(times, hostTime{host: h.Address(), ts: ts})
	}

	if len(times) < 2 {
		return nil
	}

	var minTs, maxTs int64 = times[0].ts, times[0].ts
	var minHost, maxHost string = times[0].host, times[0].host

	for _, t := range times[1:] {
		if t.ts < minTs {
			minTs = t.ts
			minHost = t.host
		}
		if t.ts > maxTs {
			maxTs = t.ts
			maxHost = t.host
		}
	}

	skew := maxTs - minTs
	if skew > maxClockSkewSeconds {
		return fmt.Errorf("clock skew of %ds detected between %s and %s (max recommended: %ds); etcd may have issues",
			skew, minHost, maxHost, maxClockSkewSeconds)
	}

	log.Debugf("clock skew OK (max %ds)", skew)
	return nil
}

// checkControllerConnectivity verifies that controllers can reach each other
// on port 2380 (etcd peering) by attempting a TCP connection test.
func (p *PreflightChecks) checkControllerConnectivity(controllers cluster.Hosts) error {
	for _, h := range controllers {
		for _, peer := range controllers {
			if h.Address() == peer.Address() {
				continue
			}

			addr := peer.PrivateAddress
			if addr == "" {
				addr = peer.Address()
			}

			// Try a simple TCP check. Use timeout to avoid hangs.
			cmd := fmt.Sprintf("timeout 3 bash -c 'echo > /dev/tcp/%s/2380' 2>/dev/null && echo ok || echo fail", addr)
			output, err := h.ExecOutput(cmd, exec.Sudo(h))
			if err != nil || strings.TrimSpace(output) != "ok" {
				return fmt.Errorf("host %s cannot reach %s on port 2380 (etcd peering)", h.Address(), addr)
			}
			log.Debugf("%s -> %s:2380 connectivity OK", h.Address(), addr)
		}
	}

	return nil
}

// humanBytes formats a byte count into a human-readable string.
func humanBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	exp := int(math.Log(float64(b)) / math.Log(1024))
	if exp > len(units) {
		exp = len(units)
	}
	val := float64(b) / math.Pow(1024, float64(exp))
	return fmt.Sprintf("%.1f %s", val, units[exp-1])
}
