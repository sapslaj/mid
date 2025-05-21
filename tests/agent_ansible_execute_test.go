package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentAnsibleExecute_happypath(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: resource.PropertyMap{
			"name": resource.NewStringProperty("command"),
			"args": resource.NewObjectProperty(resource.PropertyMap{
				"argv": resource.NewArrayProperty([]resource.PropertyValue{
					resource.NewStringProperty("echo"),
					resource.NewStringProperty("$THING"),
				}),
			}),
			"environment": resource.NewObjectProperty(resource.PropertyMap{
				"THING": resource.NewStringProperty("foo"),
			}),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewStringProperty("command"),
		res.Return["name"],
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"argv": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("echo"),
				resource.NewStringProperty("$THING"),
			}),
		}),
		res.Return["args"],
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"THING": resource.NewStringProperty("foo"),
		}),
		res.Return["environment"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["exitCode"],
	)

	result := res.Return["result"].ObjectValue()

	assert.Equal(
		t,
		resource.NewStringProperty(""),
		result["stderr"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("foo"),
		result["stdout"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		result["rc"],
	)
}

func TestAgentAnsibleExecute_invalidModuleName(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: resource.PropertyMap{
			"name": resource.NewStringProperty("404"),
			"args": resource.NewObjectProperty(resource.PropertyMap{
				"foo": resource.NewStringProperty("bar"),
			}),
		},
	})

	require.Error(t, err)

	require.Len(t, res.Failures, 0)
}

func TestAgentAnsibleExecute_invalidArguments(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: resource.PropertyMap{
			"name": resource.NewStringProperty("file"),
			"args": resource.NewObjectProperty(resource.PropertyMap{
				"foo": resource.NewStringProperty("bar"),
			}),
		},
	})

	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewStringProperty("file"),
		res.Return["name"],
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"foo": resource.NewStringProperty("bar"),
		}),
		res.Return["args"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["exitCode"],
	)

	result := res.Return["result"].ObjectValue()
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		result["failed"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("missing required arguments: path"),
		result["msg"],
	)
}

func TestAgentAnsibleExecute_failed(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:ansibleExecute"),
		Args: resource.PropertyMap{
			"name": resource.NewStringProperty("apt"),
			"args": resource.NewObjectProperty(resource.PropertyMap{
				"name": resource.NewArrayProperty([]resource.PropertyValue{
					resource.NewStringProperty("this-package-doesnt-exist"),
				}),
				"state": resource.NewStringProperty("latest"),
			}),
		},
	})

	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewStringProperty("apt"),
		res.Return["name"],
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"name": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("this-package-doesnt-exist"),
			}),
			"state": resource.NewStringProperty("latest"),
		}),
		res.Return["args"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["exitCode"],
	)

	result := res.Return["result"].ObjectValue()
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		result["failed"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("No package matching 'this-package-doesnt-exist' is available"),
		result["msg"],
	)
}
