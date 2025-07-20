package tests

import (
	"testing"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceApt(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"installs vim": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New("vim"),
				}),
				AssertCommand: "test -f /usr/bin/vim",
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"name": property.New("vim"),
					}),
					AssertCommand: "test -f /usr/bin/vim",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
			},
			AssertDeleteCommand: "test ! -f /usr/bin/vim",
		},

		"installs multiple packages": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"names": property.New([]property.Value{
						property.New("curl"),
						property.New("wget"),
					}),
					"ensure": property.New("latest"),
				}),
				AssertCommand: "test -f /usr/bin/curl && test -f /usr/bin/wget",
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"names": property.New([]property.Value{
							property.New("curl"),
							property.New("wget"),
						}),
						"ensure": property.New("latest"),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
					AssertCommand: "test -f /usr/bin/curl && test -f /usr/bin/wget",
				},
			},
			AssertDeleteCommand: "test ! -f /usr/bin/curl && test ! -f /usr/bin/wget",
		},

		"upgrade all packages": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":        property.New("*"),
					"ensure":      property.New("latest"),
					"autoremove":  property.New(true),
					"updateCache": property.New(true),
				}),
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"name":        property.New("*"),
						"ensure":      property.New("latest"),
						"autoremove":  property.New(true),
						"updateCache": property.New(true),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
			},
		},

		"apt clean": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"clean": property.New(true),
				}),
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"clean": property.New(true),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
			},
		},

		"changing the package list": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New("curl"),
				}),
				AssertCommand: "test -f /usr/bin/curl",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name": property.New("curl"),
					}),
					AssertCommand: "test -f /usr/bin/curl",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"names": property.New([]property.Value{
							property.New("curl"),
							property.New("wget"),
						}),
					}),
					AssertCommand: "test -f /usr/bin/curl && test -f /usr/bin/wget",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"name": {
								Kind:      p.Update,
								InputDiff: true,
							},
							"names": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"names": property.New([]property.Value{
							property.New("wget"),
						}),
					}),
					AssertCommand: "test ! -f /usr/bin/curl && test -f /usr/bin/wget",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"names": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name": property.New("curl"),
					}),
					AssertCommand: "test -f /usr/bin/curl && test ! -f /usr/bin/wget",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"name": {
								Kind:      p.Update,
								InputDiff: true,
							},
							"names": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
			},
		},

		"handles dpkg locking": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New("vim"),
				}),
				AssertBeforeCommand: `
					sudo apt-get update -y
					nohup sudo apt-get install emacs </dev/null >/dev/null 2>&1 & disown
				`,
				AssertCommand: `
					file /usr/bin/vim || true
					file /usr/bin/emacs || true
					test -f /usr/bin/vim
				`,
			},
		},

		"ensure absent": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":   property.New("nano"),
					"ensure": property.New("absent"),
				}),
				AssertCommand: `test ! -f /usr/bin/nano`,
			},
			AssertDeleteCommand: `test ! -f /usr/bin/nano`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			harness.AssertCommand(t, "sudo apt-get update -y && sudo apt-get install python3-apt -y")

			tc.Resource = "mid:resource:Apt"
			tc.Run(t, harness)
		})
	}
}
