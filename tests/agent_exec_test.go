package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentExec_success(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("true"),
			}),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewStringProperty("true"),
		}),
		res.Return["command"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["exitCode"],
	)
}

func TestAgentExec_failure(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("false"),
			}),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewStringProperty("false"),
		}),
		res.Return["command"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["exitCode"],
	)
}

func TestAgentExec_dir(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("touch"),
				resource.NewStringProperty("create"),
			}),
			"dir": resource.NewStringProperty("/tmp"),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewStringProperty("touch"),
			resource.NewStringProperty("create"),
		}),
		res.Return["command"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("/tmp"),
		res.Return["dir"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["exitCode"],
	)

	harness.AssertCommand(t, "test -f /tmp/create")
}

func TestAgentExec_environment(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("/bin/sh"),
				resource.NewStringProperty("-c"),
				resource.NewStringProperty("echo $OP > $FILE"),
			}),
			"environment": resource.NewObjectProperty(resource.PropertyMap{
				"FILE": resource.NewStringProperty("/tmp/environment"),
				"OP":   resource.NewStringProperty("create"),
			}),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewStringProperty("/bin/sh"),
			resource.NewStringProperty("-c"),
			resource.NewStringProperty("echo $OP > $FILE"),
		}),
		res.Return["command"],
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"FILE": resource.NewStringProperty("/tmp/environment"),
			"OP":   resource.NewStringProperty("create"),
		}),
		res.Return["environment"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["exitCode"],
	)

	harness.AssertCommand(t, "grep -q create /tmp/environment")
}

func TestAgentExec_stderrstdout(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("/bin/sh"),
				resource.NewStringProperty("-c"),
				resource.NewStringProperty("echo this is create stdout\necho this is create stderr 1>&2\n"),
			}),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewStringProperty("/bin/sh"),
			resource.NewStringProperty("-c"),
			resource.NewStringProperty("echo this is create stdout\necho this is create stderr 1>&2\n"),
		}),
		res.Return["command"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["exitCode"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("this is create stderr\n"),
		res.Return["stderr"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("this is create stdout\n"),
		res.Return["stdout"],
	)
}

func TestAgentExec_stdin(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("tee"),
				resource.NewStringProperty("/tmp/tee-stdin"),
			}),
			"stdin": resource.NewStringProperty("this is stdin\n"),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewStringProperty("tee"),
			resource.NewStringProperty("/tmp/tee-stdin"),
		}),
		res.Return["command"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("this is stdin\n"),
		res.Return["stdin"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["exitCode"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("this is stdin\n"),
		res.Return["stdout"],
	)

	harness.AssertCommand(t, "grep -q 'this is stdin' /tmp/tee-stdin")
}
