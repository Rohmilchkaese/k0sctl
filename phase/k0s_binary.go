package phase

import (
	"strconv"
	"strings"
	"time"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
)

// k0sBinaryTempFile generates a unique temporary file path for a k0s binary
// upload or download on the given host. The temp file is placed alongside the
// final binary location with a ".tmp.<timestamp>" suffix.
func k0sBinaryTempFile(h *cluster.Host) string {
	ts := strconv.Itoa(int(time.Now().UnixNano()))
	bin := h.K0sInstallLocation()
	tmp := bin + ".tmp." + ts
	if h.IsConnected() && h.IsWindows() {
		// Place the temp marker before the .exe extension
		if strings.HasSuffix(strings.ToLower(bin), ".exe") {
			tmp = strings.TrimSuffix(bin, ".exe") + ".tmp." + ts + ".exe"
		}
	}
	return tmp
}
