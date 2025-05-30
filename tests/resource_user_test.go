package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceUser(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		props  map[string]property.Value
		create string
		update string
		delete string
	}{
		"simple": {
			props: map[string]property.Value{
				"name": property.New("mid"),
			},
			create: "grep -q ^mid /etc/passwd",
			update: "grep -q ^mid /etc/passwd",
			delete: "test -z $(grep ^mid /etc/passwd)",
		},
		"manage home": {
			props: map[string]property.Value{
				"name":       property.New("mid"),
				"manageHome": property.New(true),
			},
			create: "test -d /home/mid",
			update: "test -d /home/mid",
			delete: "test ! -d /home/mid",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			t.Logf("%s: sending preview create request", name)
			_, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:User"),
				Properties: property.NewMap(tc.props),
				DryRun:     true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:User"),
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
				Urn:    MakeURN("mid:resource:User"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
				DryRun: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:User"),
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
				Urn:        MakeURN("mid:resource:User"),
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
