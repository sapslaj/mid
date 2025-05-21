package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/require"
)

func TestAgentAgentPing(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t)
	defer harness.Close()

	res, err := harness.Provider.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:agentPing"),
		Args:  resource.PropertyMap{},
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	require.Equal(t, "pong", res.Return["pong"].StringValue())
}
