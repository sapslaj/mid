package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceService(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.QEMUBackend,
	})
	defer harness.Close()

	tests := map[string]struct {
		props  resource.PropertyMap
		before string
		create string
		update string
		delete string
	}{
		"start service": {
			props: resource.PropertyMap{
				"name":   resource.NewStringProperty("cron.service"),
				"state": resource.NewStringProperty("started"),
			},
			before: "sudo systemctl disable --now cron.service && systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
			create: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'active (running)'",
			delete: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
		},
		"start and enable service": {
			props: resource.PropertyMap{
				"name":   resource.NewStringProperty("cron.service"),
				"state": resource.NewStringProperty("started"),
				"enabled": resource.NewBoolProperty(true),
			},
			before: "sudo systemctl disable --now cron.service && systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
			create: "systemctl status cron.service | grep 'cron.service; enabled' && systemctl status cron.service | grep 'active (running)'",
			delete: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
		},
		"enable service without start": {
			props: resource.PropertyMap{
				"name":   resource.NewStringProperty("cron.service"),
				"enabled": resource.NewBoolProperty(true),
			},
			before: "sudo systemctl disable --now cron.service && systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
			create: "systemctl status cron.service | grep 'cron.service; enabled' && systemctl status cron.service | grep 'inactive (dead)'",
			delete: "systemctl status cron.service | grep 'cron.service; disabled' && systemctl status cron.service | grep 'inactive (dead)'",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			if tc.before != "" {
				t.Logf("%s: running before commands", name)
				if !harness.AssertCommand(t, tc.before) {
					return
				}
			}

			t.Logf("%s: sending preview create request", name)
			_, err := harness.Provider.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Service"),
				Properties: tc.props,
				Preview:    true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Provider.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Service"),
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
				Urn:     MakeURN("mid:resource:Service"),
				Olds:    createResponse.Properties,
				News:    tc.props,
				Preview: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Provider.Update(p.UpdateRequest{
				Urn:  MakeURN("mid:resource:Service"),
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
				Urn:        MakeURN("mid:resource:Service"),
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
