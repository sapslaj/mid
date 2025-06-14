package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestAgentAnsibleExecute_happypath(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: property.NewMap(map[string]property.Value{
			"name": property.New("command"),
			"args": property.New(map[string]property.Value{
				"argv": property.New([]property.Value{
					property.New("echo"),
					property.New("$THING"),
				}),
			}),
			"environment": property.New(map[string]property.Value{
				"THING": property.New("foo"),
			}),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New("command"),
		res.Return.Get("name"),
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"argv": property.New([]property.Value{
				property.New("echo"),
				property.New("$THING"),
			}),
		}),
		res.Return.Get("args"),
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"THING": property.New("foo"),
		}),
		res.Return.Get("environment"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("exitCode"),
	)

	result := res.Return.Get("result").AsMap()

	assert.Equal(
		t,
		property.New(""),
		result.Get("stderr"),
	)
	assert.Equal(
		t,
		property.New("foo"),
		result.Get("stdout"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		result.Get("rc"),
	)
}

func TestAgentAnsibleExecute_invalidModuleName(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: property.NewMap(map[string]property.Value{
			"name": property.New("404"),
			"args": property.New(map[string]property.Value{
				"foo": property.New("bar"),
			}),
		}),
	})

	require.Error(t, err)

	require.Len(t, res.Failures, 0)
}

func TestAgentAnsibleExecute_invalidArguments(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: property.NewMap(map[string]property.Value{
			"name": property.New("file"),
			"args": property.New(map[string]property.Value{
				"foo": property.New("bar"),
			}),
		}),
	})

	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New("file"),
		res.Return.Get("name"),
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"foo": property.New("bar"),
		}),
		res.Return.Get("args"),
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("exitCode"),
	)

	result := res.Return.Get("result").AsMap()
	assert.Equal(
		t,
		property.New(true),
		result.Get("failed"),
	)
	assert.Equal(
		t,
		property.New("missing required arguments: path"),
		result.Get("msg"),
	)
}

func TestAgentAnsibleExecute_failed(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: property.NewMap(map[string]property.Value{
			"name": property.New("apt"),
			"args": property.New(map[string]property.Value{
				"name": property.New([]property.Value{
					property.New("this-package-doesnt-exist"),
				}),
				"state": property.New("latest"),
			}),
		}),
	})

	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New("apt"),
		res.Return.Get("name"),
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"name": property.New([]property.Value{
				property.New("this-package-doesnt-exist"),
			}),
			"state": property.New("latest"),
		}),
		res.Return.Get("args"),
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("exitCode"),
	)

	result := res.Return.Get("result").AsMap()
	assert.Equal(
		t,
		property.New(true),
		result.Get("failed"),
	)
	assert.Equal(
		t,
		property.New("No package matching 'this-package-doesnt-exist' is available"),
		result.Get("msg"),
	)
}
