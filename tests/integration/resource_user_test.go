package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceUser(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"simple": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New("mid"),
				}),
				AssertCommand: "grep -q ^mid /etc/passwd",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name": property.New("mid"),
					}),
					AssertCommand: "grep -q ^mid /etc/passwd",
				},
			},
			AssertDeleteCommand: "test -z $(grep ^mid /etc/passwd)",
		},

		"manage home": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"name":       property.New("mid"),
					"manageHome": property.New(true),
				}),
				AssertCommand: "test -d /home/mid",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"name":       property.New("mid"),
						"manageHome": property.New(true),
					}),
					AssertCommand: "test -d /home/mid",
				},
			},
			AssertDeleteCommand: "test ! -d /home/mid",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Resource = "mid:resource:User"
			tc.Run(t, harness)
		})
	}
}
