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
					"name":   property.New("cron.service"),
					"ensure": property.New("started"),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl disable --now cron.service
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
				AssertCommand: `set -eu
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'active (running)'
				`,
			},
			AssertDeleteCommand: `set -eu
				systemctl status cron.service | grep 'cron.service; disabled'
				systemctl status cron.service | grep 'inactive (dead)'
			`,
		},

		"start and enable service": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("cron.service"),
					"ensure":  property.New("started"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl disable --now cron.service
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
				AssertCommand: `set -eu
					systemctl status cron.service | grep 'cron.service; enabled'
					systemctl status cron.service | grep 'active (running)'
				`,
			},
			AssertDeleteCommand: `set -eu
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
					systemctl status cron.service | grep 'cron.service; disabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
				AssertCommand: `set -eu
					systemctl status cron.service | grep 'cron.service; enabled'
					systemctl status cron.service | grep 'inactive (dead)'
				`,
			},
			AssertDeleteCommand: `set -eu
				systemctl status cron.service | grep 'cron.service; disabled'
				systemctl status cron.service | grep 'inactive (dead)'
			`,
		},

		"service unit not defined during create preview": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":    property.New("mid-systemd-service-test.service"),
					"ensure":  property.New("started"),
					"enabled": property.New(true),
				}),
				AssertBeforeCommand: `set -eu
sudo rm -f /etc/systemd/system/mid-systemd-service-test.service
sudo systemctl daemon-reload
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
				AssertCommand: `set -eu
					systemctl status mid-systemd-service-test.service | grep 'mid-systemd-service-test.service; enabled'
					systemctl status mid-systemd-service-test.service | grep 'inactive (dead)'
				`,
			},
		},

		"restarts service on refresh": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":   property.New("cron.service"),
					"ensure": property.New("started"),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				AssertCommand: `set -eu
					journalctl | tail -n 10
				`,
			},
			Updates: []Operation{
				// Don't reload without refresh changes
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name":   property.New("cron.service"),
						"ensure": property.New("started"),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("1"),
							})),
						})),
					}),
					AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
					AssertCommand: `set -eu
						journalctl | tail -n 10
						journalctl | tail -n 10 | grep -v "Stopped cron.service"
						journalctl | tail -n 10 | grep -v "Started cron.service"
					`,
				},
				// Reload on refresh changes
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name":   property.New("cron.service"),
						"ensure": property.New("started"),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
					AssertCommand: `set -eu
						journalctl | tail -n 10
						journalctl | tail -n 10 | grep "Stopped cron.service"
						journalctl | tail -n 10 | grep "Started cron.service"
					`,
				},
			},
		},

		"restarts service on create if refresh triggers are defined": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":   property.New("cron.service"),
					"ensure": property.New("started"),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl start cron.service
					for i in $(seq 10); do logger space $i ; done
				`,
				AssertCommand: `set -eu
					journalctl | tail -n 10 | grep "Stopping cron.service"
					journalctl | tail -n 10 | grep "Starting cron.service"
				`,
			},
		},

		"does not restart service on create if refresh triggers are not defined": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":   property.New("cron.service"),
					"ensure": property.New("started"),
				}),
				AssertBeforeCommand: `set -eu
					sudo systemctl start cron.service
					for i in $(seq 10); do logger space $i ; done
				`,
				AssertCommand: `set -eu
					journalctl | tail -n 10 | grep -v "Stopping cron.service"
					journalctl | tail -n 10 | grep -v "Starting cron.service"
				`,
			},
		},

		"daemon-reload": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"daemonReload": property.New(true),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
				AssertCommand:       `journalctl | tail -n 10 | grep "Reloading finished in"`,
			},
			Updates: []Operation{
				// Don't reload without refresh changes
				{
					Inputs: property.NewMap(map[string]property.Value{
						"daemonReload": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("1"),
							})),
						})),
					}),
					AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
					AssertCommand:       `journalctl | tail -n 10 | grep -v "Reloading finished in"`,
				},
				// Reload on refresh changes
				{
					Inputs: property.NewMap(map[string]property.Value{
						"daemonReload": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
					AssertCommand:       `journalctl | tail -n 10 | grep "Reloading finished in"`,
				},
			},
		},

		"daemon-reexec": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"daemonReexec": property.New(true),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
				AssertCommand:       `journalctl | tail -n 10 | grep "Reexecuting."`,
			},
			Updates: []Operation{
				// Don't reload without refresh changes
				{
					Inputs: property.NewMap(map[string]property.Value{
						"daemonReexec": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("1"),
							})),
						})),
					}),
					AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
					AssertCommand:       `journalctl | tail -n 10 | grep -v "Reexecuting."`,
				},
				// Reload on refresh changes
				{
					Inputs: property.NewMap(map[string]property.Value{
						"daemonReexec": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					AssertBeforeCommand: "for i in $(seq 10); do logger space $i ; done",
					AssertCommand:       `journalctl | tail -n 10 | grep "Reexecuting."`,
				},
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
