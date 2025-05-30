package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceFile(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.QEMUBackend,
	})
	defer harness.Close()

	tests := map[string]struct {
		props  map[string]property.Value
		before string
		create string
		update string
		delete string
	}{
		"content": {
			props: map[string]property.Value{
				"path":    property.New("/foo"),
				"ensure":  property.New("file"),
				"content": property.New("bar\n"),
			},
			create: "test -f /foo && grep -q ^bar /foo",
			delete: "test ! -f /foo",
		},
		"directory": {
			props: map[string]property.Value{
				"path":   property.New("/foo"),
				"ensure": property.New("directory"),
			},
			create: "test -d /foo",
			delete: "test ! -d /foo",
		},
		"set permissions on new file": {
			props: map[string]property.Value{
				"path":    property.New("/foo"),
				"ensure":  property.New("file"),
				"content": property.New("bar\n"),
				"mode":    property.New("a=rwx"),
				"owner":   property.New("games"),
			},
			create: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777' && grep -q ^bar /foo",
			delete: "true",
		},
		"set permissions on existing file": {
			props: map[string]property.Value{
				"path":   property.New("/foo"),
				"ensure": property.New("file"),
				"mode":   property.New("a=rwx"),
				"owner":  property.New("games"),
			},
			before: "sudo touch /foo",
			create: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777'",
			delete: "true",
		},
		// FIXME: these tests are borked for some reason
		// "source asset": {
		// 	props: map[string]property.Value{
		// 		"path": property.New("/foo"),
		// 		"source": property.New(map[string]property.Value{
		// 			"a9e28acb8ab501f883219e7c9f624fb6": resource.NewAssetProperty(Must1(asset.FromPath("./resource_file_test.go"))),
		// 		}),
		// 	},
		// 	create: "grep -q 'package tests' /foo",
		// 	delete: "test ! -d /foo",
		// },
		// "source archive": {
		// 	props: map[string]property.Value{
		// 		"path":   property.New("/foo"),
		// 		"source": resource.NewArchiveProperty(Must1(archive.FromPath("./"))),
		// 	},
		// 	create: "ls -lah / && exit 1",
		// 	delete: "test ! -d /foo",
		// },
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
			_, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:File"),
				Properties: property.NewMap(tc.props),
				DryRun:     true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:File"),
				Properties: property.NewMap(tc.props),
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking create status", name)
			if !harness.AssertCommand(t, tc.create) {
				return
			}

			t.Logf("%s: sending update request", name)
			_, err = harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:File"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
				DryRun: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:File"),
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
				Urn:        MakeURN("mid:resource:File"),
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
