package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceFileLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		props        map[string]property.Value
		before       string
		beforeCreate string
		create       string
		update       string
		delete       string
	}{
		"line modification in existing file": {
			props: map[string]property.Value{
				"path":   property.New("/etc/default/motd-news"),
				"line":   property.New("ENABLED=0"),
				"regexp": property.New("^ENABLED"),
			},
			before: `cat << EOF | sudo tee /etc/default/motd-news
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
			create: "grep -q ^ENABLED=0 /etc/default/motd-news",
			update: "grep -q ^ENABLED=0 /etc/default/motd-news",
			// delete: "grep -q ^ENABLED=1 /etc/default/motd-news", // FIXME: revert on delete
		},
		"line addition to new file": {
			props: map[string]property.Value{
				"path":   property.New("/etc/default/motd-news"),
				"line":   property.New("ENABLED=0"),
				"regexp": property.New("^ENABLED"),
			},
			before: "rm -f /etc/default/motd-news",
			beforeCreate: `cat << EOF | sudo tee /etc/default/motd-news
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
			create: "grep -q ^ENABLED=0 /etc/default/motd-news",
			update: "grep -q ^ENABLED=0 /etc/default/motd-news",
			// delete: "grep -q ^ENABLED=1 /etc/default/motd-news", // FIXME: revert on delete
		},
		// TODO: "line addition to new file with create=true"
		// TODO: "line addition in existing file"
		// TODO: "line deletion in existing file"
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			if tc.before != "" {
				t.Logf("%s: running before commands", name)
				if !harness.AssertCommand(t, tc.before) {
					return
				}
			}

			t.Logf("%s: sending preview create request", name)
			_, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:FileLine"),
				Properties: property.NewMap(tc.props),
				DryRun:     true,
			})
			if !assert.NoError(t, err) {
				return
			}

			if tc.beforeCreate != "" {
				t.Logf("%s: running before create commands", name)
				if !harness.AssertCommand(t, tc.beforeCreate) {
					return
				}
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:FileLine"),
				Properties: property.NewMap(tc.props),
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking create status", name)
			if !harness.AssertCommand(t, tc.create) {
				return
			}

			t.Logf("%s: sending preview update request", name)
			_, err = harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:FileLine"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
				DryRun: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:FileLine"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
			})
			if !assert.NoError(t, err) {
				return
			}

			if tc.update == "" {
				t.Logf("%s: update check is same as create", name)
				tc.update = tc.create
			}
			t.Logf("%s: checking update status", name)
			if !harness.AssertCommand(t, tc.update) {
				return
			}

			t.Logf("%s: sending delete request", name)
			err = harness.Server.Delete(p.DeleteRequest{
				Urn:        MakeURN("mid:resource:FileLine"),
				Properties: updateResponse.Properties,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking delete status", name)
			if !harness.AssertCommand(t, tc.delete) {
				return
			}
		})
	}
}
