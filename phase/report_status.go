package phase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// ReportStatus collects and displays cluster status information.
type ReportStatus struct {
	GenericPhase

	// Writer is where the status output goes (defaults to os.Stdout via manager).
	Writer io.Writer
	// Format controls output format: "" for human-readable, "json" for JSON.
	Format string
}

// Title for the phase
func (p *ReportStatus) Title() string {
	return "Report cluster status"
}

// nodeStatus holds per-node status information.
type nodeStatus struct {
	Host             string `json:"host"`
	Role             string `json:"role"`
	Hostname         string `json:"hostname,omitempty"`
	OS               string `json:"os,omitempty"`
	K0sBinaryVersion string `json:"k0sBinaryVersion,omitempty"`
	K0sRunning       string `json:"k0sRunningVersion,omitempty"`
	DesiredVersion   string `json:"desiredVersion,omitempty"`
	NeedsUpgrade     bool   `json:"needsUpgrade"`
	ServiceRunning   bool   `json:"serviceRunning"`
	KubeletReady     bool   `json:"kubeletReady"`
	IsLeader         bool   `json:"isLeader,omitempty"`
}

// etcdMemberStatus holds etcd member information.
type etcdMemberStatus struct {
	Members []string `json:"members"`
	Healthy bool     `json:"healthy"`
	Error   string   `json:"error,omitempty"`
}

// clusterStatus is the full JSON output structure.
type clusterStatus struct {
	ClusterName    string            `json:"clusterName"`
	DesiredVersion string            `json:"desiredVersion,omitempty"`
	Nodes          []nodeStatus      `json:"nodes"`
	Etcd           *etcdMemberStatus `json:"etcd,omitempty"`
}

// Run the phase
func (p *ReportStatus) Run(_ context.Context) error {
	cs := clusterStatus{
		ClusterName: p.Config.Metadata.Name,
	}

	if p.Config.Spec.K0s != nil && p.Config.Spec.K0s.Version != nil {
		cs.DesiredVersion = p.Config.Spec.K0s.Version.String()
	}

	for _, h := range p.Config.Spec.Hosts {
		ns := p.collectNodeStatus(h)
		cs.Nodes = append(cs.Nodes, ns)
	}

	cs.Etcd = p.collectEtcdStatus()

	if p.Format == "json" {
		return p.outputJSON(cs)
	}

	return p.outputHuman(cs)
}

func (p *ReportStatus) collectNodeStatus(h *cluster.Host) nodeStatus {
	ns := nodeStatus{
		Host:         h.Address(),
		Role:         h.Role,
		NeedsUpgrade: h.Metadata.NeedsUpgrade,
		IsLeader:     h.Metadata.IsK0sLeader,
		KubeletReady: h.Metadata.Ready,
	}

	if h.Metadata.Hostname != "" {
		ns.Hostname = h.Metadata.Hostname
	}

	if h.OSVersion != nil {
		ns.OS = h.OSVersion.String()
	}

	if h.Metadata.K0sBinaryVersion != nil {
		ns.K0sBinaryVersion = h.Metadata.K0sBinaryVersion.String()
	}

	if h.Metadata.K0sRunningVersion != nil {
		ns.K0sRunning = h.Metadata.K0sRunningVersion.String()
	}

	if p.Config.Spec.K0s != nil && p.Config.Spec.K0s.Version != nil {
		ns.DesiredVersion = p.Config.Spec.K0s.Version.String()
	}

	// Check if the k0s service is running
	if h.Configurer != nil {
		svcName := h.K0sServiceName()
		ns.ServiceRunning = h.Configurer.ServiceIsRunning(h, svcName)
	}

	// For controllers, check API reachability
	if h.IsController() && h.Configurer != nil && ns.ServiceRunning {
		apiURL := p.Config.Spec.NodeInternalKubeAPIURL(h)
		if err := h.CheckHTTPStatus(apiURL, 200, 401, 403); err == nil {
			// API is reachable (401/403 still means the API is up, just auth differs)
			log.Debugf("%s: API server is reachable at %s", h, apiURL)
		} else {
			log.Debugf("%s: API server check failed: %v", h, err)
		}
	}

	// For workers with running k0s, check kubelet ready status via leader
	if !h.IsController() && h.Metadata.K0sRunningVersion != nil {
		leader := p.Config.Spec.K0sLeader()
		if leader != nil && leader.Configurer != nil {
			output, err := leader.ExecOutput(
				leader.Configurer.KubectlCmdf(leader, leader.K0sDataDir(),
					"get node -l kubernetes.io/hostname=%s -o jsonpath='{.items[0].status.conditions[?(@.type==\"Ready\")].status}'",
					strings.ToLower(h.Metadata.Hostname)),
				exec.HideOutput(), exec.Sudo(leader),
			)
			if err == nil && strings.TrimSpace(strings.Trim(output, "'")) == "True" {
				ns.KubeletReady = true
			}
		}
	}

	return ns
}

func (p *ReportStatus) collectEtcdStatus() *etcdMemberStatus {
	es := &etcdMemberStatus{}

	if len(p.Config.Metadata.EtcdMembers) > 0 {
		es.Members = p.Config.Metadata.EtcdMembers
	}

	// Try to get etcd health from the leader
	leader := p.Config.Spec.K0sLeader()
	if leader == nil || leader.Configurer == nil || leader.Metadata.K0sRunningVersion == nil {
		es.Error = "no running controller found to query etcd"
		return es
	}

	output, err := leader.ExecOutput(
		leader.Configurer.K0sCmdf("etcd status --data-dir=%s", leader.K0sDataDir()),
		exec.Sudo(leader),
	)
	if err != nil {
		// etcd status command may not exist in older versions, try alternative
		log.Debugf("%s: etcd status failed: %v, trying member-list", leader, err)
		if len(es.Members) > 0 {
			es.Healthy = true // if we got members earlier, etcd was at least partially working
		} else {
			es.Error = fmt.Sprintf("failed to query etcd status: %v", err)
		}
		return es
	}

	// Try to parse as JSON
	var statusResult map[string]any
	if err := json.Unmarshal([]byte(output), &statusResult); err == nil {
		es.Healthy = true
	} else {
		// Even non-JSON output means the command ran
		if strings.TrimSpace(output) != "" {
			es.Healthy = true
		}
	}

	return es
}

func (p *ReportStatus) outputJSON(cs clusterStatus) error {
	enc := json.NewEncoder(p.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(cs)
}

func (p *ReportStatus) outputHuman(cs clusterStatus) error {
	w := tabwriter.NewWriter(p.Writer, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Cluster: %s\n", cs.ClusterName)
	if cs.DesiredVersion != "" {
		fmt.Fprintf(w, "Desired k0s version: %s\n", cs.DesiredVersion)
	}
	fmt.Fprintf(w, "\n")

	// Node table
	fmt.Fprintf(w, "HOST\tROLE\tHOSTNAME\tK0S VERSION\tSTATUS\tNOTES\n")
	fmt.Fprintf(w, "----\t----\t--------\t-----------\t------\t-----\n")

	for _, n := range cs.Nodes {
		ver := n.K0sRunning
		if ver == "" {
			ver = n.K0sBinaryVersion
			if ver != "" {
				ver += " (not running)"
			} else {
				ver = "(not installed)"
			}
		}

		status := statusIcon(n.ServiceRunning)
		if n.KubeletReady {
			status += " Ready"
		} else if n.ServiceRunning {
			if n.Role == "controller" {
				status += " Running"
			} else {
				status += " Not Ready"
			}
		} else {
			status += " Stopped"
		}

		var notes []string
		if n.IsLeader {
			notes = append(notes, "leader")
		}
		if n.NeedsUpgrade {
			notes = append(notes, "needs upgrade")
		}
		if n.DesiredVersion != "" && n.K0sRunning != "" && n.K0sRunning != n.DesiredVersion {
			notes = append(notes, fmt.Sprintf("want %s", n.DesiredVersion))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			n.Host,
			n.Role,
			n.Hostname,
			ver,
			status,
			strings.Join(notes, ", "),
		)
	}

	fmt.Fprintf(w, "\n")

	// Etcd status
	if cs.Etcd != nil {
		if cs.Etcd.Healthy {
			fmt.Fprintf(w, "etcd: healthy (%d members)\n", len(cs.Etcd.Members))
		} else if cs.Etcd.Error != "" {
			fmt.Fprintf(w, "etcd: %s\n", cs.Etcd.Error)
		} else {
			fmt.Fprintf(w, "etcd: unknown\n")
		}
		if len(cs.Etcd.Members) > 0 {
			fmt.Fprintf(w, "etcd members: %s\n", strings.Join(cs.Etcd.Members, ", "))
		}
	}

	fmt.Fprintf(w, "\n")

	return w.Flush()
}

func statusIcon(running bool) string {
	if running {
		return "[+]"
	}
	return "[-]"
}
