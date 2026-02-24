package phase

import (
	"testing"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig"
	"github.com/stretchr/testify/require"
)

func newControllerHost(sshAddr, privateAddr string) *cluster.Host {
	h := &cluster.Host{
		Connection: rig.Connection{
			SSH: &rig.SSH{Address: sshAddr},
		},
		Role:           "controller",
		PrivateAddress: privateAddr,
	}
	return h
}

func TestFindOrphanedMembers(t *testing.T) {
	t.Run("no etcd members returns nil", func(t *testing.T) {
		p := &ReconcileEtcdMembers{
			GenericPhase: GenericPhase{
				Config: &v1beta1.Cluster{
					Metadata: &v1beta1.ClusterMetadata{
						EtcdMembers: nil,
					},
					Spec: &cluster.Spec{
						Hosts: cluster.Hosts{
							newControllerHost("10.0.0.1", "192.168.1.1"),
						},
					},
				},
			},
		}
		require.Nil(t, p.findOrphanedMembers())
	})

	t.Run("all members match controllers", func(t *testing.T) {
		p := &ReconcileEtcdMembers{
			GenericPhase: GenericPhase{
				Config: &v1beta1.Cluster{
					Metadata: &v1beta1.ClusterMetadata{
						EtcdMembers: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
					},
					Spec: &cluster.Spec{
						Hosts: cluster.Hosts{
							newControllerHost("10.0.0.1", "192.168.1.1"),
							newControllerHost("10.0.0.2", "192.168.1.2"),
							newControllerHost("10.0.0.3", "192.168.1.3"),
						},
					},
				},
			},
		}
		require.Nil(t, p.findOrphanedMembers())
	})

	t.Run("orphaned member detected", func(t *testing.T) {
		p := &ReconcileEtcdMembers{
			GenericPhase: GenericPhase{
				Config: &v1beta1.Cluster{
					Metadata: &v1beta1.ClusterMetadata{
						EtcdMembers: []string{"192.168.1.1", "192.168.1.2", "192.168.1.99"},
					},
					Spec: &cluster.Spec{
						Hosts: cluster.Hosts{
							newControllerHost("10.0.0.1", "192.168.1.1"),
							newControllerHost("10.0.0.2", "192.168.1.2"),
						},
					},
				},
			},
		}
		orphaned := p.findOrphanedMembers()
		require.Equal(t, []string{"192.168.1.99"}, orphaned)
	})

	t.Run("match via public address when no private address", func(t *testing.T) {
		p := &ReconcileEtcdMembers{
			GenericPhase: GenericPhase{
				Config: &v1beta1.Cluster{
					Metadata: &v1beta1.ClusterMetadata{
						EtcdMembers: []string{"10.0.0.1", "10.0.0.2"},
					},
					Spec: &cluster.Spec{
						Hosts: cluster.Hosts{
							newControllerHost("10.0.0.1", ""),
							newControllerHost("10.0.0.2", ""),
						},
					},
				},
			},
		}
		require.Nil(t, p.findOrphanedMembers())
	})

	t.Run("multiple orphaned members", func(t *testing.T) {
		p := &ReconcileEtcdMembers{
			GenericPhase: GenericPhase{
				Config: &v1beta1.Cluster{
					Metadata: &v1beta1.ClusterMetadata{
						EtcdMembers: []string{"192.168.1.1", "192.168.1.50", "192.168.1.51"},
					},
					Spec: &cluster.Spec{
						Hosts: cluster.Hosts{
							newControllerHost("10.0.0.1", "192.168.1.1"),
						},
					},
				},
			},
		}
		orphaned := p.findOrphanedMembers()
		require.Equal(t, []string{"192.168.1.50", "192.168.1.51"}, orphaned)
	})

	t.Run("worker hosts are not considered", func(t *testing.T) {
		worker := newControllerHost("10.0.0.2", "192.168.1.2")
		worker.Role = "worker"
		p := &ReconcileEtcdMembers{
			GenericPhase: GenericPhase{
				Config: &v1beta1.Cluster{
					Metadata: &v1beta1.ClusterMetadata{
						EtcdMembers: []string{"192.168.1.1", "192.168.1.2"},
					},
					Spec: &cluster.Spec{
						Hosts: cluster.Hosts{
							newControllerHost("10.0.0.1", "192.168.1.1"),
							worker,
						},
					},
				},
			},
		}
		// 192.168.1.2 is a worker, not a controller, so it's orphaned from etcd's perspective
		orphaned := p.findOrphanedMembers()
		require.Equal(t, []string{"192.168.1.2"}, orphaned)
	})
}
