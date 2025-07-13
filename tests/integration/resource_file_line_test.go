package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceFileLine(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"line modification in existing file": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/etc/default/motd-news"),
					"line":   property.New("ENABLED=0"),
					"regexp": property.New("^ENABLED"),
				}),
				AssertBeforeCommand: `cat << EOF | sudo tee /etc/default/motd-news
# Enable/disable the dynamic MOTD news service
# This is a useful way to provide dynamic, informative
# information pertinent to the users and administrators
# of the local system
ENABLED=1

# Configure the source of dynamic MOTD news
# White space separated list of 0 to many news services
# For security reasons, these must be https
# and have a valid certificate
# Canonical runs a service at motd.ubuntu.com, and you
# can easily run one too
URLS=""

# Specify the time in seconds, you're willing to wait for
# dynamic MOTD news
# Note that news messages are fetched in the background by
# a systemd timer, so this should never block boot or login
WAIT=5
EOF
`,
				AssertCommand: "grep -q ^ENABLED=0 /etc/default/motd-news",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/etc/default/motd-news"),
						"line":   property.New("ENABLED=0"),
						"regexp": property.New("^ENABLED"),
					}),
					AssertCommand: "grep -q ^ENABLED=0 /etc/default/motd-news",
				},
			},
		},

		"line addition in existing file": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/etc/default/motd-news"),
					"line":   property.New("ENABLED=0"),
					"regexp": property.New("^ENABLED"),
				}),
				AssertBeforeCommand: `cat << EOF | sudo tee /etc/default/motd-news
# Configure the source of dynamic MOTD news
# White space separated list of 0 to many news services
# For security reasons, these must be https
# and have a valid certificate
# Canonical runs a service at motd.ubuntu.com, and you
# can easily run one too
URLS=""

# Specify the time in seconds, you're willing to wait for
# dynamic MOTD news
# Note that news messages are fetched in the background by
# a systemd timer, so this should never block boot or login
WAIT=5
EOF
`,
				AssertCommand: "grep -q ^ENABLED=0 /etc/default/motd-news",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/etc/default/motd-news"),
						"line":   property.New("ENABLED=0"),
						"regexp": property.New("^ENABLED"),
					}),
					AssertCommand: "grep -q ^ENABLED=0 /etc/default/motd-news",
				},
			},
		},

		"line deletion in existing file": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/etc/default/motd-news"),
					"regexp": property.New("^ENABLED"),
					"ensure": property.New("absent"),
				}),
				AssertBeforeCommand: `cat << EOF | sudo tee /etc/default/motd-news
# Enable/disable the dynamic MOTD news service
# This is a useful way to provide dynamic, informative
# information pertinent to the users and administrators
# of the local system
ENABLED=1

# Configure the source of dynamic MOTD news
# White space separated list of 0 to many news services
# For security reasons, these must be https
# and have a valid certificate
# Canonical runs a service at motd.ubuntu.com, and you
# can easily run one too
URLS=""

# Specify the time in seconds, you're willing to wait for
# dynamic MOTD news
# Note that news messages are fetched in the background by
# a systemd timer, so this should never block boot or login
WAIT=5
EOF
`,
				AssertCommand: "! grep -q ^ENABLED /etc/default/motd-news",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/etc/default/motd-news"),
						"regexp": property.New("^ENABLED"),
						"ensure": property.New("absent"),
					}),
					AssertCommand: "! grep -q ^ENABLED /etc/default/motd-news",
				},
			},
		},

		"line addition and updates to new file with create=true": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/fileline"),
					"line":   property.New("foo bar baz"),
					"regexp": property.New("^foo"),
					"create": property.New(true),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				AssertCommand: "grep -F 'foo bar baz' /fileline",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar baz"),
						"regexp": property.New("^foo"),
						"create": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					AssertCommand: "grep -F 'foo bar baz' /fileline",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"triggers": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar baz"),
						"regexp": property.New("^foo"),
						"create": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					AssertCommand: "grep -F 'foo bar baz' /fileline",
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
			},
		},

		"line updates": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/fileline"),
					"line":   property.New("foo bar baz"),
					"regexp": property.New("^foo"),
				}),
				AssertBeforeCommand: "sudo touch /fileline",
				AssertCommand:       "grep -F 'foo bar baz' /fileline",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar foo"),
						"regexp": property.New("^foo"),
					}),
					AssertCommand: "grep -F 'foo bar foo' /fileline",
					ExpectedDiff: &p.DiffResponse{
						HasChanges:          true,
						DeleteBeforeReplace: true,
						DetailedDiff: map[string]p.PropertyDiff{
							"line": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar foo"),
						"regexp": property.New("^foo"),
					}),
					AssertCommand: "grep -F 'foo bar foo' /fileline",
					ExpectedDiff: &p.DiffResponse{
						HasChanges:          false,
						DeleteBeforeReplace: true,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar foo"),
						"regexp": property.New("^foo"),
					}),
					Refresh:       true,
					AssertCommand: "grep -F 'foo bar foo' /fileline",
					ExpectedDiff: &p.DiffResponse{
						HasChanges:          false,
						DeleteBeforeReplace: true,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar baz"),
						"regexp": property.New("^foo"),
					}),
					AssertCommand: "grep -F 'foo bar baz' /fileline",
					ExpectedDiff: &p.DiffResponse{
						HasChanges:          true,
						DeleteBeforeReplace: true,
						DetailedDiff: map[string]p.PropertyDiff{
							"line": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/fileline"),
						"line":   property.New("foo bar baz"),
						"regexp": property.New("^foo"),
					}),
					AssertBeforeCommand: `
						set -eux
						cat /fileline
						sudo sed -i 's/foo bar/foo foo/' /fileline
						cat /fileline
					`,
					Refresh:       true,
					AssertCommand: "grep -F 'foo bar baz' /fileline",
					ExpectedDiff: &p.DiffResponse{
						HasChanges:          true,
						DeleteBeforeReplace: true,
						DetailedDiff: map[string]p.PropertyDiff{
							"line": {
								Kind:      p.Update,
								InputDiff: false,
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Resource = "mid:resource:FileLine"
			tc.Run(t, harness)
		})
	}
}
