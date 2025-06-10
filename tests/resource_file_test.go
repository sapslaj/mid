package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceFile(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.QEMUBackend,
	})
	defer harness.Close()

	tests := map[string]LifeCycleTest{
		"content": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":    property.New("/foo"),
					"ensure":  property.New("file"),
					"content": property.New("bar\n"),
				}),
				AssertCommand: "test -f /foo && grep -q ^bar /foo",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":    property.New("/foo"),
						"ensure":  property.New("file"),
						"content": property.New("bar\n"),
					}),
					AssertCommand: "test -f /foo && grep -q ^bar /foo",
				},
			},
			AssertDeleteCommand: "test ! -f /foo",
		},
		"directory": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/foo"),
					"ensure": property.New("directory"),
				}),
				AssertCommand: "test -d /foo",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/foo"),
						"ensure": property.New("directory"),
					}),
					AssertCommand: "test -d /foo",
				},
			},
			AssertDeleteCommand: "test ! -d /foo",
		},
		"set permissions on new file": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":    property.New("/foo"),
					"ensure":  property.New("file"),
					"content": property.New("bar\n"),
					"mode":    property.New("a=rwx"),
					"owner":   property.New("games"),
				}),
				AssertCommand: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777' && grep -q ^bar /foo",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":    property.New("/foo"),
						"ensure":  property.New("file"),
						"content": property.New("bar\n"),
						"mode":    property.New("a=rwx"),
						"owner":   property.New("games"),
					}),
					AssertCommand: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777' && grep -q ^bar /foo",
				},
			},
		},
		"set permissions on existing file": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":   property.New("/foo"),
					"ensure": property.New("file"),
					"mode":   property.New("a=rwx"),
					"owner":  property.New("games"),
				}),
				AssertBeforeCommand: "sudo touch /foo",
				AssertCommand:       "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777'",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/foo"),
						"ensure": property.New("file"),
						"mode":   property.New("a=rwx"),
						"owner":  property.New("games"),
					}),
					AssertCommand: "stat -c '%n %U %a' /foo && test \"$(stat -c '%n %U %a' /foo)\" = '/foo games 777'",
				},
			},
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
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			tc.Resource = "mid:resource:File"

			tc.Run(t, harness)
		})
	}
}
