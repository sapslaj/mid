package tests

import (
	"testing"

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
		"line addition to new file": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/etc/default/motd-news"),
					"line":   property.New("ENABLED=0"),
					"regexp": property.New("^ENABLED"),
				}),
				AssertBeforeCommand: `rm -f /etc/default/motd-news
cat << EOF | sudo tee /etc/default/motd-news
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
		// TODO: "line addition to new file with create=true"
		// TODO: "line addition in existing file"
		// TODO: "line deletion in existing file"
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.Resource = "mid:resource:FileLine"

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Run(t, harness)
		})
	}
}
