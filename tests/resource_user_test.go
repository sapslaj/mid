package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceUser(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	tests := map[string]struct {
		props  resource.PropertyMap
		create string
		update string
		delete string
	}{
		"simple": {
			props: resource.PropertyMap{
				"name": resource.NewStringProperty("mid"),
			},
			create: "grep -q ^mid /etc/passwd",
			update: "grep -q ^mid /etc/passwd",
			delete: "test -z $(grep ^mid /etc/passwd)",
		},
		"manage home": {
			props: resource.PropertyMap{
				"name":       resource.NewStringProperty("mid"),
				"manageHome": resource.NewBoolProperty(true),
			},
			create: "test -d /home/mid",
			update: "test -d /home/mid",
			delete: "test ! -d /home/mid",
		},
	}

	for name, tc := range tests {
		t.Logf("%s: sending preview create request", name)
		_, err := harness.Provider.Create(p.CreateRequest{
			Urn:        MakeURN("mid:resource:User"),
			Properties: tc.props,
			Preview:    true,
		})
		if !assert.NoError(t, err) {
			continue
		}

		t.Logf("%s: sending create request", name)
		createResponse, err := harness.Provider.Create(p.CreateRequest{
			Urn:        MakeURN("mid:resource:User"),
			Properties: tc.props,
		})
		if !assert.NoError(t, err) {
			continue
		}

		t.Logf("%s: checking create status", name)
		if !harness.AssertCommand(t, tc.create) {
			continue
		}

		t.Logf("%s: sending preview update request", name)
		_, err = harness.Provider.Update(p.UpdateRequest{
			Urn:     MakeURN("mid:resource:User"),
			Olds:    createResponse.Properties,
			News:    tc.props,
			Preview: true,
		})
		if !assert.NoError(t, err) {
			continue
		}

		t.Logf("%s: sending update request", name)
		updateResponse, err := harness.Provider.Update(p.UpdateRequest{
			Urn:  MakeURN("mid:resource:User"),
			Olds: createResponse.Properties,
			News: tc.props,
		})
		if !assert.NoError(t, err) {
			continue
		}

		if tc.update == "" {
			t.Logf("%s: update check is same as create", name)
			tc.update = tc.create
		}
		t.Logf("%s: checking update status", name)
		if !harness.AssertCommand(t, tc.update) {
			continue
		}

		t.Logf("%s: sending delete request", name)
		err = harness.Provider.Delete(p.DeleteRequest{
			Urn:        MakeURN("mid:resource:User"),
			Properties: updateResponse.Properties,
		})
		if !assert.NoError(t, err) {
			continue
		}

		t.Logf("%s: checking delete status", name)
		if !harness.AssertCommand(t, tc.delete) {
			continue
		}
	}
}
