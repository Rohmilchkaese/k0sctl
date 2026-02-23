package cluster

import (
	"testing"

	"github.com/k0sproject/dig"
	"github.com/k0sproject/rig"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestUnmarshalYAML_EmptyK0s(t *testing.T) {
	t.Run("empty k0s key does not leave K0s nil", func(t *testing.T) {
		specYaml := []byte(`
hosts:
  - role: controller
    ssh:
      address: 10.0.0.1
k0s:
`)
		spec := &Spec{}
		err := yaml.Unmarshal(specYaml, spec)
		require.NoError(t, err)
		require.NotNil(t, spec.K0s, "K0s should not be nil after unmarshaling empty k0s: key")
	})

	t.Run("omitted k0s key does not leave K0s nil", func(t *testing.T) {
		specYaml := []byte(`
hosts:
  - role: controller
    ssh:
      address: 10.0.0.1
`)
		spec := &Spec{}
		err := yaml.Unmarshal(specYaml, spec)
		require.NoError(t, err)
		require.NotNil(t, spec.K0s, "K0s should not be nil when k0s key is omitted")
	})
}

func TestNodeInternalKubeAPIURL_NilK0s(t *testing.T) {
	spec := &Spec{
		Hosts: Hosts{
			&Host{
				Role: "controller",
				Connection: rig.Connection{
					SSH: &rig.SSH{
						Address: "10.0.0.1",
					},
				},
			},
		},
	}
	// Should not panic even with nil K0s
	require.Equal(t, "https://127.0.0.1:6443", spec.NodeInternalKubeAPIURL(spec.Hosts[0]))
}

func TestKubeAPIURL(t *testing.T) {
	t.Run("with external address and port", func(t *testing.T) {
		spec := &Spec{
			K0s: &K0s{
				Config: dig.Mapping(map[string]any{
					"spec": dig.Mapping(map[string]any{
						"api": dig.Mapping(map[string]any{
							"port":            6444,
							"externalAddress": "test.example.com",
						}),
					}),
				}),
			}, Hosts: Hosts{
				&Host{
					Role: "controller",
					Connection: rig.Connection{
						SSH: &rig.SSH{
							Address: "10.0.0.1",
						},
					},
				},
			},
		}
		require.Equal(t, "https://test.example.com:6444", spec.KubeAPIURL())
	})

	t.Run("without k0s config", func(t *testing.T) {
		spec := &Spec{
			Hosts: Hosts{
				&Host{
					Role:           "controller",
					PrivateAddress: "10.0.0.1",
					Connection: rig.Connection{
						SSH: &rig.SSH{
							Address: "192.168.0.1",
						},
					},
				},
			},
		}
		require.Equal(t, "https://192.168.0.1:6443", spec.KubeAPIURL())
	})

	t.Run("with CPLB", func(t *testing.T) {
		specYaml := []byte(`
hosts:
  - role: controller
    ssh:
      address: 192.168.0.1
    privateAddress: 10.0.0.1
k0s:
  config:
    spec:
      network:
        controlPlaneLoadBalancing:
          enabled: true
          type: Keepalived
          keepalived:
            vrrpInstances:
            - virtualIPs: ["192.168.0.10/24"]
              authPass: CPLB
            virtualServers:
            - ipAddress: 192.168.0.10`)

		spec := &Spec{}
		err := yaml.Unmarshal(specYaml, spec)
		require.NoError(t, err)

		require.Equal(t, "https://192.168.0.10:6443", spec.KubeAPIURL())
	})
}
