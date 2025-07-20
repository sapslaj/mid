package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestAgentAgentPing(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	res, err := harness.Server.Invoke(p.InvokeRequest{
		Token: tokens.Type("mid:agent:agentPing"),
		Args:  property.NewMap(map[string]property.Value{}),
	})
	require.NoError(t, err)

	require.Len(t, res.Failures, 0)

	require.Equal(t, property.New("pong"), res.Return.Get("pong"))
}
