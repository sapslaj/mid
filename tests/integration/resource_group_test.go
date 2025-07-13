package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceGroup(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"simple": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New("mid"),
				}),
				AssertCommand: "grep -q ^mid /etc/group",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name": property.New("mid"),
					}),
					AssertCommand: "grep -q ^mid /etc/group",
				},
			},
			AssertDeleteCommand: "test -z $(grep ^mid /etc/group)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Resource = "mid:resource:Group"
			tc.Run(t, harness)
		})
	}
}
