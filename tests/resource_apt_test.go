package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceApt(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	tests := map[string]LifeCycleTest{
		"installs vim": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New("vim"),
				}),
				AssertCommand: "test -f /usr/bin/vim",
			},
			AssertDeleteCommand: "test ! -f /usr/bin/vim",
		},
		"installs multiple packages": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"names": property.New([]property.Value{
						property.New("curl"),
						property.New("wget"),
					}),
					"ensure": property.New("latest"),
				}),
				AssertCommand: "test -f /usr/bin/curl && test -f /usr/bin/wget",
			},
			AssertDeleteCommand: "test ! -f /usr/bin/curl && test ! -f /usr/bin/wget",
		},
		"upgrade all packages": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":        property.New("*"),
					"ensure":      property.New("latest"),
					"autoremove":  property.New(true),
					"updateCache": property.New(true),
				}),
			},
		},
		"apt clean": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"clean": property.New(true),
				}),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			tc.Resource = "mid:resource:Apt"
			tc.Run(t, harness)
		})
	}
}
