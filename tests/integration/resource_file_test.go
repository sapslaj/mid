package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceFile(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"content": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":    property.New("/foo"),
					"ensure":  property.New("file"),
					"content": property.New("bar\n"),
				}),
				AssertCommand: `set -eu
					test -f /foo
					grep -q ^bar /foo
				`,
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"path":    property.New("/foo"),
						"ensure":  property.New("file"),
						"content": property.New("bar\n"),
					}),
					AssertCommand: `set -eu
						test -f /foo
						grep -q ^bar /foo
					`,
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
					Refresh: true,
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
				AssertCommand: `set -eu
					stat -c '%n %U %a' /foo
					test "$(stat -c '%n %U %a' /foo)" = '/foo games 777'
					grep -q ^bar /foo
				`,
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"path":    property.New("/foo"),
						"ensure":  property.New("file"),
						"content": property.New("bar\n"),
						"mode":    property.New("a=rwx"),
						"owner":   property.New("games"),
					}),
					AssertCommand: `set -eu
						stat -c '%n %U %a' /foo
						test "$(stat -c '%n %U %a' /foo)" = '/foo games 777'
						grep -q ^bar /foo
					`,
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
				AssertCommand: `set -eu
					stat -c '%n %U %a' /foo
					test "$(stat -c '%n %U %a' /foo)" = '/foo games 777'
				`,
			},
			Updates: []Operation{
				{
					Refresh: true,
					Inputs: property.NewMap(map[string]property.Value{
						"path":   property.New("/foo"),
						"ensure": property.New("file"),
						"mode":   property.New("a=rwx"),
						"owner":  property.New("games"),
					}),
					AssertCommand: `set -eu
						stat -c '%n %U %a' /foo
						test "$(stat -c '%n %U %a' /foo)" = '/foo games 777'
					`,
				},
			},
		},

		"allows for the parent directory to not be created yet during dry run": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":    property.New("/nested/sub/dir/foo"),
					"ensure":  property.New("file"),
					"content": property.New("bar\n"),
				}),
				AssertBeforeCommand:      "test ! -d /nested/sub/dir",
				AssertAfterDryRunCommand: "sudo mkdir -p /nested/sub/dir",
				AssertCommand:            "test -f /nested/sub/dir/foo",
			},
		},

		"path replace": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":    property.New("/foo"),
					"ensure":  property.New("file"),
					"content": property.New("bar\n"),
				}),
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":    property.New("/bar"),
						"ensure":  property.New("file"),
						"content": property.New("bar\n"),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: false,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"path": {
								Kind:      p.UpdateReplace,
								InputDiff: true,
							},
						},
					},
					AssertInMiddleOfReplaceCommand: `set -eu
						ls -lah /foo /bar || true
						test -f /foo
						test -f /bar
					`,
					AssertCommand: `set -eu
						ls -lah /foo /bar || true
						test ! -f /foo
						test -f /bar
						cat /bar
						grep -q ^bar /bar
					`,
				},
			},
		},

		"read should detect changes": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"path":    property.New("/foo"),
					"ensure":  property.New("file"),
					"content": property.New("bar\n"),
				}),
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"path":    property.New("/foo"),
						"ensure":  property.New("file"),
						"content": property.New("bar\n"),
					}),
					AssertBeforeCommand: "echo baz | sudo tee -a /foo",
					Refresh:             true,
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: false,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"content": {
								Kind:      p.Update,
								InputDiff: false,
							},
						},
					},
					AssertCommand: `set -eu
						ls -lah /foo || true
						test -f /foo
						grep -q ^bar /foo
					`,
				},
			},
		},

		// FIXME: these tests are borked for some reason
		// "source asset": {
		// 	Create: Operation{
		// 		Inputs: property.NewMap(map[string]property.Value{
		// 			"path":   property.New("/foo"),
		// 			"source": property.New(must.Must1(asset.FromPath("./resource_file_test.go"))),
		// 		}),
		// 		AssertCommand: "grep -q 'package tests' /foo",
		// 	},
		// 	AssertDeleteCommand: "test ! -d /foo",
		// },

		// "source archive": {
		// 	Create: Operation{
		// 		Inputs: property.NewMap(map[string]property.Value{
		// 			"path":   property.New("/foo"),
		// 			"source": property.New(must.Must1(archive.FromPath("./"))),
		// 		}),
		// 		AssertCommand: "ls -lah / && exit 1",
		// 	},
		// 	AssertDeleteCommand: "test ! -d /foo",
		// },
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.Resource = "mid:resource:File"

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Run(t, harness)
		})
	}
}
