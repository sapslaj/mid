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

func TestAgentFileStat_regfile(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: property.NewMap(map[string]property.Value{
			"path": property.New("/etc/shadow"),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("followSymlinks"),
	)
	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("calculateChecksum"),
	)
	assert.Equal(
		t,
		property.New("/etc/shadow"),
		res.Return.Get("path"),
	)
	assert.Equal(
		t,
		property.New(true),
		res.Return.Get("exists"),
	)
	assert.Equal(
		t,
		property.New("shadow"),
		res.Return.Get("baseName"),
	)
	assert.Greater(
		t,
		res.Return.Get("size").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"isDir":     property.New(false),
			"isRegular": property.New(true),
			"int":       property.New(float64(416)),
			"octal":     property.New("640"),
			"string":    property.New("-rw-r-----"),
		}),
		res.Return.Get("fileMode"),
	)
	assert.True(t, res.Return.Get("modifiedTime").IsString())
	assert.True(t, res.Return.Get("accessTime").IsString())
	assert.True(t, res.Return.Get("createTime").IsString())
	assert.Greater(
		t,
		res.Return.Get("dev").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(42)),
		res.Return.Get("gid"),
	)
	assert.Equal(
		t,
		property.New("shadow"),
		res.Return.Get("groupName"),
	)
	assert.Greater(
		t,
		res.Return.Get("inode").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("nlink"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("uid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("userName"),
	)
}

func TestAgentFileStat_symlinks(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: property.NewMap(map[string]property.Value{
			"path": property.New("/bin"),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("followSymlinks"),
	)
	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("calculateChecksum"),
	)
	assert.Equal(
		t,
		property.New("/bin"),
		res.Return.Get("path"),
	)
	assert.Equal(
		t,
		property.New(true),
		res.Return.Get("exists"),
	)
	assert.Equal(
		t,
		property.New("bin"),
		res.Return.Get("baseName"),
	)
	assert.Greater(
		t,
		res.Return.Get("size").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"isDir":     property.New(false),
			"isRegular": property.New(false),
			"int":       property.New(float64(134218239)),
			"octal":     property.New("1000000777"),
			"string":    property.New("Lrwxrwxrwx"),
		}),
		res.Return.Get("fileMode"),
	)
	assert.True(t, res.Return.Get("modifiedTime").IsString())
	assert.True(t, res.Return.Get("accessTime").IsString())
	assert.True(t, res.Return.Get("createTime").IsString())
	assert.Greater(
		t,
		res.Return.Get("dev").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("gid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("groupName"),
	)
	assert.Greater(
		t,
		res.Return.Get("inode").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("nlink"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("uid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("userName"),
	)

	res, err = harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: property.NewMap(map[string]property.Value{
			"path":           property.New("/bin"),
			"followSymlinks": property.New(true),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New(true),
		res.Return.Get("followSymlinks"),
	)
	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("calculateChecksum"),
	)
	assert.Equal(
		t,
		property.New("/bin"),
		res.Return.Get("path"),
	)
	assert.Equal(
		t,
		property.New(true),
		res.Return.Get("exists"),
	)
	assert.Equal(
		t,
		property.New("bin"),
		res.Return.Get("baseName"),
	)
	assert.Greater(
		t,
		res.Return.Get("size").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"isDir":     property.New(true),
			"isRegular": property.New(false),
			"int":       property.New(float64(2147484141)),
			"octal":     property.New("20000000755"),
			"string":    property.New("drwxr-xr-x"),
		}),
		res.Return.Get("fileMode"),
	)
	assert.True(t, res.Return.Get("modifiedTime").IsString())
	assert.True(t, res.Return.Get("accessTime").IsString())
	assert.True(t, res.Return.Get("createTime").IsString())
	assert.Greater(
		t,
		res.Return.Get("dev").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("gid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("groupName"),
	)
	assert.Greater(
		t,
		res.Return.Get("inode").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("nlink"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("uid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("userName"),
	)
}

func TestAgentFileStat_nonexistant(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: property.NewMap(map[string]property.Value{
			"path": property.New("/404-not-found"),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("followSymlinks"),
	)
	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("calculateChecksum"),
	)
	assert.Equal(
		t,
		property.New("/404-not-found"),
		res.Return.Get("path"),
	)
	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("exists"),
	)
	assert.True(t, res.Return.Get("baseName").IsNull())
	assert.True(t, res.Return.Get("size").IsNull())
	assert.True(t, res.Return.Get("fileMode").IsNull())
	assert.True(t, res.Return.Get("modifiedTime").IsNull())
	assert.True(t, res.Return.Get("accessTime").IsNull())
	assert.True(t, res.Return.Get("createTime").IsNull())
	assert.True(t, res.Return.Get("dev").IsNull())
	assert.True(t, res.Return.Get("gid").IsNull())
	assert.True(t, res.Return.Get("groupName").IsNull())
	assert.True(t, res.Return.Get("inode").IsNull())
	assert.True(t, res.Return.Get("nlink").IsNull())
	assert.True(t, res.Return.Get("uid").IsNull())
	assert.True(t, res.Return.Get("userName").IsNull())
}

func TestAgentFileStat_checksum(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	require.True(t, harness.AssertCommand(t, "echo foo | sudo tee /foo"))

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: property.NewMap(map[string]property.Value{
			"path":              property.New("/foo"),
			"calculateChecksum": property.New(true),
		}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		property.New(false),
		res.Return.Get("followSymlinks"),
	)
	assert.Equal(
		t,
		property.New(true),
		res.Return.Get("calculateChecksum"),
	)
	assert.Equal(
		t,
		property.New("/foo"),
		res.Return.Get("path"),
	)
	assert.Equal(
		t,
		property.New(true),
		res.Return.Get("exists"),
	)
	assert.Equal(
		t,
		property.New("foo"),
		res.Return.Get("baseName"),
	)
	assert.Equal(
		t,
		property.New(float64(4)),
		res.Return.Get("size"),
	)
	assert.Equal(
		t,
		property.New(map[string]property.Value{
			"isDir":     property.New(false),
			"isRegular": property.New(true),
			"int":       property.New(float64(420)),
			"octal":     property.New("644"),
			"string":    property.New("-rw-r--r--"),
		}),
		res.Return.Get("fileMode"),
	)
	assert.True(t, res.Return.Get("modifiedTime").IsString())
	assert.True(t, res.Return.Get("accessTime").IsString())
	assert.True(t, res.Return.Get("createTime").IsString())
	assert.Greater(
		t,
		res.Return.Get("dev").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("gid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("groupName"),
	)
	assert.Greater(
		t,
		res.Return.Get("inode").AsNumber(),
		1.0,
	)
	assert.Equal(
		t,
		property.New(float64(1)),
		res.Return.Get("nlink"),
	)
	assert.Equal(
		t,
		property.New(float64(0)),
		res.Return.Get("uid"),
	)
	assert.Equal(
		t,
		property.New("root"),
		res.Return.Get("userName"),
	)
	assert.Equal(
		t,
		property.New("b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"),
		res.Return.Get("sha256Checksum"),
	)
}
