package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceSystemdService(t *testing.T) {
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
					"ensure": property.New("started"),
				}),
				AssertBeforeCommand: "sudo systemctl disable --now cron.service && systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
				AssertCommand:       "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'active (running)'",
			},
			AssertDeleteCommand: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
		},
		"start and enable service": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("cron.service"),
					"ensure":   property.New("started"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: "sudo systemctl disable --now cron.service && systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
				AssertCommand:       "systemctl status cron.service | grep 'cron.service; enabled' && systemctl status cron.service | grep 'active (running)'",
			},
			AssertDeleteCommand: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
		},
		"enable service without start": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("cron.service"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: "sudo systemctl disable --now cron.service && systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
				AssertCommand:       "systemctl status cron.service | grep 'cron.service; enabled' && systemctl status cron.service | grep 'inactive (dead)'",
			},
			AssertDeleteCommand: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
		},
		"service unit not defined during create preview": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("mid-systemd-service-test.service"),
					"ensure":  property.New("started"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: `sudo rm -f /etc/systemd/system/mid-systemd-service-test.service ; sudo systemctl daemon-reload
cat << EOF | sudo tee /etc/systemd/system/mid-systemd-service-test.service
[Unit]
Description=systemd service test
[Service]
Type=oneshot
ExecStart=/usr/bin/echo test
[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reload
sudo systemctl disable --now mid-systemd-service-test.service
`,
				AssertCommand: "systemctl status mid-systemd-service-test.service ; systemctl status mid-systemd-service-test.service | grep 'mid-systemd-service-test.service; enabled' && systemctl status mid-systemd-service-test.service | grep 'inactive (dead)'",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			tc.Resource = "mid:resource:SystemdService"

			tc.Run(t, harness)
		})
	}
}
