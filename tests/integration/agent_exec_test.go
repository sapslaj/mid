package tests

import (
	"testing"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestAgentExec_success(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: property.NewMap(map[string]property.Value{
			"command": property.New([]property.Value{
				property.New("true"),
			}),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New([]property.Value{
			property.New("true"),
		}),
		res.Return.Get("command"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("exitCode"),
	)
}

func TestAgentExec_failure(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: property.NewMap(map[string]property.Value{
			"command": property.New([]property.Value{
				property.New("false"),
			}),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New([]property.Value{
			property.New("false"),
		}),
		res.Return.Get("command"),
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("exitCode"),
	)
}

func TestAgentExec_dir(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: property.NewMap(map[string]property.Value{
			"command": property.New([]property.Value{
				property.New("touch"),
				property.New("create"),
			}),
			"dir": property.New("/tmp"),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New([]property.Value{
			property.New("touch"),
			property.New("create"),
		}),
		res.Return.Get("command"),
	)
	assert.Equal(
		t,
		property.New("/tmp"),
		res.Return.Get("dir"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("exitCode"),
	)

	harness.AssertCommand(t, "test -f /tmp/create")
}

func TestAgentExec_environment(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: property.NewMap(map[string]property.Value{
			"command": property.New([]property.Value{
				property.New("/bin/sh"),
				property.New("-c"),
				property.New("echo $OP > $FILE"),
			}),
			"environment": property.New(map[string]property.Value{
				"FILE": property.New("/tmp/environment"),
				"OP":   property.New("create"),
			}),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New([]property.Value{
			property.New("/bin/sh"),
			property.New("-c"),
			property.New("echo $OP > $FILE"),
		}),
		res.Return.Get("command"),
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"FILE": property.New("/tmp/environment"),
			"OP":   property.New("create"),
		}),
		res.Return.Get("environment"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("exitCode"),
	)

	harness.AssertCommand(t, "grep -q create /tmp/environment")
}

func TestAgentExec_stderrstdout(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: property.NewMap(map[string]property.Value{
			"command": property.New([]property.Value{
				property.New("/bin/sh"),
				property.New("-c"),
				property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
			}),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New([]property.Value{
			property.New("/bin/sh"),
			property.New("-c"),
			property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
		}),
		res.Return.Get("command"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("exitCode"),
	)
	assert.Equal(
		t,
		property.New("this is create stderr\n"),
		res.Return.Get("stderr"),
	)
	assert.Equal(
		t,
		property.New("this is create stdout\n"),
		res.Return.Get("stdout"),
	)
}

func TestAgentExec_stdin(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:exec"),
		Args: property.NewMap(map[string]property.Value{
			"command": property.New([]property.Value{
				property.New("tee"),
				property.New("/tmp/tee-stdin"),
			}),
			"stdin": property.New("this is stdin\n"),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New([]property.Value{
			property.New("tee"),
			property.New("/tmp/tee-stdin"),
		}),
		res.Return.Get("command"),
	)
	assert.Equal(
		t,
		property.New("this is stdin\n"),
		res.Return.Get("stdin"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("exitCode"),
	)
	assert.Equal(
		t,
		property.New("this is stdin\n"),
		res.Return.Get("stdout"),
	)

	harness.AssertCommand(t, "grep -q 'this is stdin' /tmp/tee-stdin")
}
