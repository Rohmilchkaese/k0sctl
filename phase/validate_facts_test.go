package phase

import (
	"testing"

	"github.com/k0sproject/dig"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestValidateNodeLocalLoadBalancing(t *testing.T) {
	baseConfig := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{{Role: "single"}},
			K0s:   &cluster.K0s{Config: dig.Mapping{}},
		},
	}

	p := &ValidateFacts{GenericPhase: GenericPhase{Config: baseConfig}}

	t.Run("fails when enabled on single", func(t *testing.T) {
		baseConfig.Spec.K0s.Config["network"] = dig.Mapping{
			"nodeLocalLoadBalancing": dig.Mapping{
				"enabled": true,
			},
		}
		err := p.validateNodeLocalLoadBalancing()
		require.ErrorContains(t, err, "spec.k0s.config.network.nodeLocalLoadBalancing.enabled")
	})

	t.Run("passes when disabled on single", func(t *testing.T) {
		baseConfig.Spec.K0s.Config["network"] = dig.Mapping{
			"nodeLocalLoadBalancing": dig.Mapping{
				"enabled": false,
			},
		}
		require.NoError(t, p.validateNodeLocalLoadBalancing())
	})

	t.Run("passes when not single", func(t *testing.T) {
		baseConfig.Spec.Hosts[0].Role = "controller"
		baseConfig.Spec.K0s.Config["network"] = dig.Mapping{
			"nodeLocalLoadBalancing": dig.Mapping{
				"enabled": true,
			},
		}
		require.NoError(t, p.validateNodeLocalLoadBalancing())
	})
}

// testLogHook captures log entries for assertion in tests.
type testLogHook struct {
	entries []*log.Entry
}

func (h *testLogHook) Levels() []log.Level { return log.AllLevels }
func (h *testLogHook) Fire(e *log.Entry) error {
	h.entries = append(h.entries, e)
	return nil
}

func (h *testLogHook) hasWarningContaining(s string) bool {
	for _, e := range h.entries {
		if e.Level == log.WarnLevel && contains(e.Message, s) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestWarnMultiControllerWithoutLoadBalancer(t *testing.T) {
	setupHook := func() *testLogHook {
		hook := &testLogHook{}
		log.AddHook(hook)
		return hook
	}

	t.Run("no warning with single controller", func(t *testing.T) {
		hook := setupHook()
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{{Role: "controller"}},
				K0s:   &cluster.K0s{Config: dig.Mapping{}},
			},
		}
		p := &ValidateFacts{GenericPhase: GenericPhase{Config: cfg}}
		p.warnMultiControllerWithoutLoadBalancer()
		require.False(t, hook.hasWarningContaining("multi-controller"))
	})

	t.Run("warns with multiple controllers and no LB config", func(t *testing.T) {
		hook := setupHook()
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{{Role: "controller"}, {Role: "controller"}},
				K0s:   &cluster.K0s{Config: dig.Mapping{}},
			},
		}
		p := &ValidateFacts{GenericPhase: GenericPhase{Config: cfg}}
		p.warnMultiControllerWithoutLoadBalancer()
		require.True(t, hook.hasWarningContaining("multi-controller"))
	})

	t.Run("no warning with externalAddress set", func(t *testing.T) {
		hook := setupHook()
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{{Role: "controller"}, {Role: "controller"}},
				K0s: &cluster.K0s{Config: dig.Mapping{
					"spec": dig.Mapping{
						"api": dig.Mapping{"externalAddress": "10.0.0.1"},
					},
				}},
			},
		}
		p := &ValidateFacts{GenericPhase: GenericPhase{Config: cfg}}
		p.warnMultiControllerWithoutLoadBalancer()
		require.False(t, hook.hasWarningContaining("multi-controller"))
	})

	t.Run("no warning with NLLB enabled", func(t *testing.T) {
		hook := setupHook()
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{{Role: "controller"}, {Role: "controller"}},
				K0s: &cluster.K0s{Config: dig.Mapping{
					"spec": dig.Mapping{
						"network": dig.Mapping{
							"nodeLocalLoadBalancing": dig.Mapping{"enabled": true},
						},
					},
				}},
			},
		}
		p := &ValidateFacts{GenericPhase: GenericPhase{Config: cfg}}
		p.warnMultiControllerWithoutLoadBalancer()
		require.False(t, hook.hasWarningContaining("multi-controller"))
	})

	t.Run("no warning with CPLB enabled", func(t *testing.T) {
		hook := setupHook()
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{{Role: "controller"}, {Role: "controller"}},
				K0s: &cluster.K0s{Config: dig.Mapping{
					"spec": dig.Mapping{
						"network": dig.Mapping{
							"controlPlaneLoadBalancing": dig.Mapping{"enabled": true},
						},
					},
				}},
			},
		}
		p := &ValidateFacts{GenericPhase: GenericPhase{Config: cfg}}
		p.warnMultiControllerWithoutLoadBalancer()
		require.False(t, hook.hasWarningContaining("multi-controller"))
	})

	t.Run("warns with nil K0s config", func(t *testing.T) {
		hook := setupHook()
		cfg := &v1beta1.Cluster{
			Spec: &cluster.Spec{
				Hosts: cluster.Hosts{{Role: "controller"}, {Role: "controller"}},
			},
		}
		p := &ValidateFacts{GenericPhase: GenericPhase{Config: cfg}}
		p.warnMultiControllerWithoutLoadBalancer()
		require.True(t, hook.hasWarningContaining("multi-controller"))
	})
}
