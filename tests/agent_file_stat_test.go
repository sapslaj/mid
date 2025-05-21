package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentFileStat_regfile(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: resource.PropertyMap{
			"path": resource.NewStringProperty("/etc/shadow"),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["followSymlinks"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["calculateChecksum"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("/etc/shadow"),
		res.Return["path"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		res.Return["exists"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("shadow"),
		res.Return["baseName"],
	)
	assert.Greater(
		t,
		res.Return["size"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"isDir":     resource.NewBoolProperty(false),
			"isRegular": resource.NewBoolProperty(true),
			"int":       resource.NewNumberProperty(416),
			"octal":     resource.NewStringProperty("640"),
			"string":    resource.NewStringProperty("-rw-r-----"),
		}),
		res.Return["fileMode"],
	)
	assert.True(t, res.Return["modifiedTime"].HasValue())
	assert.True(t, res.Return["accessTime"].HasValue())
	assert.True(t, res.Return["createTime"].HasValue())
	assert.Greater(
		t,
		res.Return["dev"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(42),
		res.Return["gid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("shadow"),
		res.Return["groupName"],
	)
	assert.Greater(
		t,
		res.Return["inode"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["nlink"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["uid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["userName"],
	)
}

func TestAgentFileStat_symlinks(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: resource.PropertyMap{
			"path": resource.NewStringProperty("/bin"),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["followSymlinks"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["calculateChecksum"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("/bin"),
		res.Return["path"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		res.Return["exists"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("bin"),
		res.Return["baseName"],
	)
	assert.Greater(
		t,
		res.Return["size"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"isDir":     resource.NewBoolProperty(false),
			"isRegular": resource.NewBoolProperty(false),
			"int":       resource.NewNumberProperty(134218239),
			"octal":     resource.NewStringProperty("1000000777"),
			"string":    resource.NewStringProperty("Lrwxrwxrwx"),
		}),
		res.Return["fileMode"],
	)
	assert.True(t, res.Return["modifiedTime"].HasValue())
	assert.True(t, res.Return["accessTime"].HasValue())
	assert.True(t, res.Return["createTime"].HasValue())
	assert.Greater(
		t,
		res.Return["dev"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["gid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["groupName"],
	)
	assert.Greater(
		t,
		res.Return["inode"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["nlink"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["uid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["userName"],
	)

	res, err = harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: resource.PropertyMap{
			"path":           resource.NewStringProperty("/bin"),
			"followSymlinks": resource.NewBoolProperty(true),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		res.Return["followSymlinks"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["calculateChecksum"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("/bin"),
		res.Return["path"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		res.Return["exists"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("bin"),
		res.Return["baseName"],
	)
	assert.Greater(
		t,
		res.Return["size"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"isDir":     resource.NewBoolProperty(true),
			"isRegular": resource.NewBoolProperty(false),
			"int":       resource.NewNumberProperty(2147484141),
			"octal":     resource.NewStringProperty("20000000755"),
			"string":    resource.NewStringProperty("drwxr-xr-x"),
		}),
		res.Return["fileMode"],
	)
	assert.True(t, res.Return["modifiedTime"].HasValue())
	assert.True(t, res.Return["accessTime"].HasValue())
	assert.True(t, res.Return["createTime"].HasValue())
	assert.Greater(
		t,
		res.Return["dev"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["gid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["groupName"],
	)
	assert.Greater(
		t,
		res.Return["inode"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["nlink"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["uid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["userName"],
	)
}

func TestAgentFileStat_nonexistant(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: resource.PropertyMap{
			"path": resource.NewStringProperty("/404-not-found"),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["followSymlinks"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["calculateChecksum"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("/404-not-found"),
		res.Return["path"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["exists"],
	)
	assert.True(t, res.Return["baseName"].IsNull())
	assert.True(t, res.Return["size"].IsNull())
	assert.True(t, res.Return["fileMode"].IsNull())
	assert.True(t, res.Return["modifiedTime"].IsNull())
	assert.True(t, res.Return["accessTime"].IsNull())
	assert.True(t, res.Return["createTime"].IsNull())
	assert.True(t, res.Return["dev"].IsNull())
	assert.True(t, res.Return["gid"].IsNull())
	assert.True(t, res.Return["groupName"].IsNull())
	assert.True(t, res.Return["inode"].IsNull())
	assert.True(t, res.Return["nlink"].IsNull())
	assert.True(t, res.Return["uid"].IsNull())
	assert.True(t, res.Return["userName"].IsNull())
}

func TestAgentFileStat_checksum(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	require.True(t, harness.AssertCommand(t, "echo foo > /foo"))

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:fileStat"),
		Args: resource.PropertyMap{
			"path":              resource.NewStringProperty("/foo"),
			"calculateChecksum": resource.NewBoolProperty(true),
		},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	assert.Equal(
		t,
		resource.NewBoolProperty(false),
		res.Return["followSymlinks"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		res.Return["calculateChecksum"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("/foo"),
		res.Return["path"],
	)
	assert.Equal(
		t,
		resource.NewBoolProperty(true),
		res.Return["exists"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("foo"),
		res.Return["baseName"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(4),
		res.Return["size"],
	)
	assert.Equal(
		t,
		resource.NewObjectProperty(resource.PropertyMap{
			"isDir":     resource.NewBoolProperty(false),
			"isRegular": resource.NewBoolProperty(true),
			"int":       resource.NewNumberProperty(420),
			"octal":     resource.NewStringProperty("644"),
			"string":    resource.NewStringProperty("-rw-r--r--"),
		}),
		res.Return["fileMode"],
	)
	assert.True(t, res.Return["modifiedTime"].HasValue())
	assert.True(t, res.Return["accessTime"].HasValue())
	assert.True(t, res.Return["createTime"].HasValue())
	assert.Greater(
		t,
		res.Return["dev"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["gid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["groupName"],
	)
	assert.Greater(
		t,
		res.Return["inode"].NumberValue(),
		1.0,
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(1),
		res.Return["nlink"],
	)
	assert.Equal(
		t,
		resource.NewNumberProperty(0),
		res.Return["uid"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("root"),
		res.Return["userName"],
	)
	assert.Equal(
		t,
		resource.NewStringProperty("b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"),
		res.Return["sha256Checksum"],
	)
}
