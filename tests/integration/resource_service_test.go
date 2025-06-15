package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceService(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.QEMUBackend,
	})
	defer harness.Close()

	tests := map[string]LifeCycleTest{
		"start service": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":  property.New("cron.service"),
					"state": property.New("started"),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl disable --now cron.service
					systemctl status cron.service || true
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
				AssertCommand: `set -eu
					systemctl status cron.service || true
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'active (running)'
				`,
			},
			AssertDeleteCommand: `set -eu
				systemctl status cron.service || true
				systemctl status cron.service | grep 'cron.service; disabled'
				systemctl status cron.service | grep 'inactive (dead)'
			`,
		},

		"start and enable service": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("cron.service"),
					"state":   property.New("started"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl disable --now cron.service
					systemctl status cron.service || true
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
				AssertCommand: `set -eu
					systemctl status cron.service || true
					systemctl status cron.service | grep 'cron.service; enabled'
					systemctl status cron.service | grep 'active (running)'
				`,
			},
			AssertDeleteCommand: `set -eu
				systemctl status cron.service || true
				systemctl status cron.service | grep 'cron.service; disabled'
				systemctl status cron.service | grep 'inactive (dead)'
			`,
		},

		"enable service without start": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("cron.service"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl disable --now cron.service
					systemctl status cron.service || true
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
				AssertCommand: `set -eu
					systemctl status cron.service || true
					systemctl status cron.service | grep 'cron.service; enabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
			},
			AssertDeleteCommand: `set -eu
				systemctl status cron.service || true
				systemctl status cron.service | grep 'cron.service; disabled'
				systemctl status cron.service | grep 'inactive (dead)'
			`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			tc.Resource = "mid:resource:Service"

			tc.Run(t, harness)
		})
	}
}
