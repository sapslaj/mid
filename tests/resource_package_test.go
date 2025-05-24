package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourcePackage(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.QEMUBackend,
	})
	defer harness.Close()

	tests := map[string]struct {
		props  resource.PropertyMap
		create string
		update string
		delete string
	}{
		"installs vim": {
			props: resource.PropertyMap{
				"name": resource.NewStringProperty("vim"),
			},
			create: "test -f /usr/bin/vim",
			delete: "test ! -f /usr/bin/vim",
		},
		"installs multiple packages": {
			props: resource.PropertyMap{
				"names": resource.NewArrayProperty([]resource.PropertyValue{
					resource.NewStringProperty("curl"),
					resource.NewStringProperty("wget"),
				}),
				"ensure": resource.NewStringProperty("latest"),
			},
			create: "test -f /usr/bin/curl && test -f /usr/bin/wget",
			delete: "test ! -f /usr/bin/curl && test ! -f /usr/bin/wget",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			t.Logf("%s: sending preview create request", name)
			_, err := harness.Provider.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Package"),
				Properties: tc.props,
				Preview:    true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Provider.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Package"),
				Properties: tc.props,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking create status", name)
			if !harness.AssertCommand(t, tc.create) {
				return
			}

			t.Logf("%s: sending preview update request", name)
			_, err = harness.Provider.Update(p.UpdateRequest{
				Urn:     MakeURN("mid:resource:Package"),
				Olds:    createResponse.Properties,
				News:    tc.props,
				Preview: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Provider.Update(p.UpdateRequest{
				Urn:  MakeURN("mid:resource:Package"),
				Olds: createResponse.Properties,
				News: tc.props,
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
			err = harness.Provider.Delete(p.DeleteRequest{
				Urn:        MakeURN("mid:resource:Package"),
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
