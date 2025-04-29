package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	// "github.com/pulumi/pulumi/sdk/v3/go/common/resource/archive"
	// "github.com/pulumi/pulumi/sdk/v3/go/common/resource/asset"
	"github.com/stretchr/testify/assert"
)

func TestResourceFile(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	tests := map[string]struct {
		props  resource.PropertyMap
		before string
		create string
		update string
		delete string
	}{
		"content": {
			props: resource.PropertyMap{
				"path":    resource.NewStringProperty("/foo"),
				"ensure":  resource.NewStringProperty("file"),
				"content": resource.NewStringProperty("bar\n"),
			},
			create: "test -f /foo && grep -q ^bar /foo",
			delete: "test ! -f /foo",
		},
		"directory": {
			props: resource.PropertyMap{
				"path":   resource.NewStringProperty("/foo"),
				"ensure": resource.NewStringProperty("directory"),
			},
			create: "test -d /foo",
			delete: "test ! -d /foo",
		},
		"set permissions on new file": {
			props: resource.PropertyMap{
				"path":    resource.NewStringProperty("/foo"),
				"ensure":  resource.NewStringProperty("file"),
				"content": resource.NewStringProperty("bar\n"),
				"mode":    resource.NewStringProperty("a=rwx"),
				"owner":   resource.NewStringProperty("games"),
			},
			create: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777' && grep -q ^bar /foo",
			delete: "true",
		},
		"set permissions on existing file": {
			props: resource.PropertyMap{
				"path":   resource.NewStringProperty("/foo"),
				"ensure": resource.NewStringProperty("file"),
				"mode":   resource.NewStringProperty("a=rwx"),
				"owner":  resource.NewStringProperty("games"),
			},
			before: "touch /foo",
			create: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777'",
			delete: "true",
		},
		// FIXME: these tests are borked for some reason
		// "source asset": {
		// 	props: resource.PropertyMap{
		// 		"path": resource.NewStringProperty("/foo"),
		// 		"source": resource.NewObjectProperty(resource.PropertyMap{
		// 			"a9e28acb8ab501f883219e7c9f624fb6": resource.NewAssetProperty(Must1(asset.FromPath("./resource_file_test.go"))),
		// 		}),
		// 	},
		// 	create: "grep -q 'package tests' /foo",
		// 	delete: "test ! -d /foo",
		// },
		// "source archive": {
		// 	props: resource.PropertyMap{
		// 		"path":   resource.NewStringProperty("/foo"),
		// 		"source": resource.NewArchiveProperty(Must1(archive.FromPath("./"))),
		// 	},
		// 	create: "ls -lah / && exit 1",
		// 	delete: "test ! -d /foo",
		// },
	}

	for name, tc := range tests {
		if tc.before != "" {
			t.Logf("%s: running before commands", name)
			if !harness.AssertCommand(t, tc.before) {
				continue
			}
		}

		t.Logf("%s: sending create request", name)
		createResponse, err := harness.Provider.Create(p.CreateRequest{
			Urn:        MakeURN("mid:resource:File"),
			Properties: tc.props,
		})
		if !assert.NoError(t, err) {
			continue
		}

		t.Logf("%s: checking create status", name)
		if !harness.AssertCommand(t, tc.create) {
			continue
		}

		t.Logf("%s: sending update request", name)
		updateResponse, err := harness.Provider.Update(p.UpdateRequest{
			Urn:  MakeURN("mid:resource:File"),
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
			Urn:        MakeURN("mid:resource:File"),
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
