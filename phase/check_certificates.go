package phase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// CertInfo holds information about a single certificate file.
type CertInfo struct {
	Host    string
	Path    string
	Subject string
	Expiry  time.Time
	IsCA    bool
}

// CheckCertificates checks certificate expiry on controllers.
//
// For each running controller, it lists all .crt files in the k0s PKI
// directory and checks their expiry using openssl. Results are stored
// in the Results field and logged with appropriate severity.
type CheckCertificates struct {
	GenericPhase

	controllers cluster.Hosts

	// Results holds certificate information collected during Run.
	Results []CertInfo

	// ExpiryWarningDays is the threshold (in days) for warning about
	// soon-to-expire certificates. Defaults to 30.
	ExpiryWarningDays int
}

// Title for the phase
func (p *CheckCertificates) Title() string {
	return "Check certificates"
}

// Prepare the phase
func (p *CheckCertificates) Prepare(config *v1beta1.Cluster) error {
	p.Config = config
	p.controllers = p.Config.Spec.Hosts.Controllers().Filter(func(h *cluster.Host) bool {
		return h.Metadata.K0sRunningVersion != nil && h.IsConnected()
	})
	if p.ExpiryWarningDays == 0 {
		p.ExpiryWarningDays = 30
	}
	return nil
}

// ShouldRun when there are running controllers to check.
func (p *CheckCertificates) ShouldRun() bool {
	return len(p.controllers) > 0
}

// Run the phase
func (p *CheckCertificates) Run(ctx context.Context) error {
	for _, h := range p.controllers {
		if err := ctx.Err(); err != nil {
			return err
		}
		certs, err := p.checkHost(h)
		if err != nil {
			log.Warnf("%s: failed to check certificates: %v", h, err)
			continue
		}
		p.Results = append(p.Results, certs...)
	}

	now := time.Now()
	var expired, expiringSoon int
	for _, c := range p.Results {
		remaining := c.Expiry.Sub(now)
		days := int(remaining.Hours() / 24)

		switch {
		case remaining <= 0:
			log.Warnf("%s: certificate %s (subject: %s) EXPIRED %d days ago",
				c.Host, c.Path, c.Subject, -days)
			expired++
		case days <= p.ExpiryWarningDays:
			log.Warnf("%s: certificate %s (subject: %s) expires in %d days (%s)",
				c.Host, c.Path, c.Subject, days, c.Expiry.Format("2006-01-02"))
			expiringSoon++
		default:
			log.Infof("%s: certificate %s OK - expires %s (%d days)",
				c.Host, c.Path, c.Expiry.Format("2006-01-02"), days)
		}
	}

	if expired > 0 {
		log.Warnf("%d certificate(s) have EXPIRED", expired)
	}
	if expiringSoon > 0 {
		log.Warnf("%d certificate(s) will expire within %d days", expiringSoon, p.ExpiryWarningDays)
	}
	if expired == 0 && expiringSoon == 0 && len(p.Results) > 0 {
		log.Infof("all %d certificate(s) are valid", len(p.Results))
	}

	return nil
}

func (p *CheckCertificates) checkHost(h *cluster.Host) ([]CertInfo, error) {
	pkiDir := h.Configurer.HostPath(h.K0sDataDir()) + "/pki"

	log.Infof("%s: checking certificates in %s", h, pkiDir)

	// Find all .crt files in the PKI directory and its subdirectories.
	output, err := h.ExecOutput(
		fmt.Sprintf("find %s -name '*.crt' -type f 2>/dev/null || true", pkiDir),
		exec.Sudo(h),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		log.Debugf("%s: no certificates found in %s", h, pkiDir)
		return nil, nil
	}

	var certs []CertInfo
	for _, certPath := range strings.Split(strings.TrimSpace(output), "\n") {
		certPath = strings.TrimSpace(certPath)
		if certPath == "" {
			continue
		}

		certOutput, err := h.ExecOutput(
			fmt.Sprintf("openssl x509 -in %s -noout -enddate -subject 2>/dev/null", certPath),
			exec.Sudo(h),
		)
		if err != nil {
			log.Debugf("%s: failed to read certificate %s: %v", h, certPath, err)
			continue
		}

		// Check if it's a CA certificate.
		caOutput, _ := h.ExecOutput(
			fmt.Sprintf("openssl x509 -in %s -noout -ext basicConstraints 2>/dev/null", certPath),
			exec.Sudo(h),
		)

		cert := parseCertOutput(h.String(), certPath, certOutput, caOutput)
		if cert != nil {
			certs = append(certs, *cert)
		}
	}

	return certs, nil
}

func parseCertOutput(host, certPath, output, caOutput string) *CertInfo {
	info := &CertInfo{
		Host: host,
		Path: certPath,
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if after, ok := strings.CutPrefix(line, "notAfter="); ok {
			// OpenSSL date format: "Jan  2 15:04:05 2006 GMT"
			// The double-space handles single-digit days.
			t, err := time.Parse("Jan  2 15:04:05 2006 GMT", after)
			if err != nil {
				t, err = time.Parse("Jan 2 15:04:05 2006 GMT", after)
			}
			if err == nil {
				info.Expiry = t
			}
		}

		if after, ok := strings.CutPrefix(line, "subject="); ok {
			info.Subject = strings.TrimSpace(after)
		}
	}

	if strings.Contains(caOutput, "CA:TRUE") {
		info.IsCA = true
	}

	if info.Expiry.IsZero() {
		return nil
	}

	return info
}
