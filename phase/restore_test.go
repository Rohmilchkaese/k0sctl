package phase

import (
	"testing"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/version"
	"github.com/stretchr/testify/require"
)

func TestRestorePrepare(t *testing.T) {
	t.Run("rejects restore on running cluster", func(t *testing.T) {
		runningVersion := version.MustParse("v1.28.0+k0s.0")
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{
					{
						Role:       "controller",
						Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.1"}},
						Metadata: cluster.HostMetadata{
							K0sRunningVersion: runningVersion,
						},
					},
				},
				K0s: &cluster.K0s{Version: version.MustParse("v1.28.0+k0s.0")},
			},
		}

		p := &Restore{RestoreFrom: "/tmp/backup.tar.gz"}
		err := p.Prepare(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot restore backup on a running cluster")
	})

	t.Run("allows restore on fresh host", func(t *testing.T) {
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{
					{
						Role:       "controller",
						Connection: rig.Connection{SSH: &rig.SSH{Address: "10.0.0.1"}},
						Metadata:   cluster.HostMetadata{},
					},
				},
				K0s: &cluster.K0s{Version: version.MustParse("v1.28.0+k0s.0")},
			},
		}

		p := &Restore{RestoreFrom: "/tmp/backup.tar.gz"}
		err := p.Prepare(cfg)
		require.NoError(t, err)
	})

	t.Run("skips when no restore path", func(t *testing.T) {
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				K0s: &cluster.K0s{Version: version.MustParse("v1.28.0+k0s.0")},
			},
		}

		p := &Restore{RestoreFrom: ""}
		err := p.Prepare(cfg)
		require.NoError(t, err)
	})
}

func TestRestoreShouldRun(t *testing.T) {
	t.Run("false when cluster is running", func(t *testing.T) {
		runningVersion := version.MustParse("v1.28.0+k0s.0")
		p := &Restore{
			RestoreFrom: "/tmp/backup.tar.gz",
			leader: &cluster.Host{
				Metadata: cluster.HostMetadata{
					K0sRunningVersion: runningVersion,
				},
			},
		}
		require.False(t, p.ShouldRun())
	})

	t.Run("false when host is marked for reset", func(t *testing.T) {
		p := &Restore{
			RestoreFrom: "/tmp/backup.tar.gz",
			leader: &cluster.Host{
				Reset: true,
			},
		}
		require.False(t, p.ShouldRun())
	})

	t.Run("true on fresh host with restore path", func(t *testing.T) {
		p := &Restore{
			RestoreFrom: "/tmp/backup.tar.gz",
			leader:      &cluster.Host{},
		}
		require.True(t, p.ShouldRun())
	})
}
