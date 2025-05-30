package tests

import (
	"strings"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/pkg/telemetry"
	mid "github.com/sapslaj/mid/provider"
	"github.com/sapslaj/mid/tests/testmachine"
)

func MakeURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(typ), "name")
}

func Must1[A any](a A, err error) A {
	if err != nil {
		panic(err)
	}
	return a
}

type ProviderTestHarness struct {
	TestMachine *testmachine.TestMachine
	Provider    p.Provider
	Server      integration.Server
	Telemetry   *telemetry.TelemetryStuff
}

func NewProviderTestHarness(t *testing.T, tmConfig testmachine.Config) *ProviderTestHarness {
	t.Helper()

	var err error
	harness := &ProviderTestHarness{}

	t.Log("starting telemetry")
	harness.Telemetry = telemetry.StartTelemetry(t.Context())

	t.Log("creating new test machine")
	harness.TestMachine, err = testmachine.New(t, tmConfig)
	require.NoError(t, err)

	t.Log("creating and configuring provider")
	harness.Provider, err = mid.Provider()
	require.NoError(t, err)

	harness.Server, err = integration.NewServer(
		t.Context(),
		mid.Name,
		semver.MustParse("1.0.0"),
		integration.WithProvider(harness.Provider),
	)
	require.NoError(t, err)
	err = harness.Server.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			"connection": property.New(map[string]property.Value{
				"user":     property.New(harness.TestMachine.SSHUsername),
				"password": property.New(harness.TestMachine.SSHPassword),
				"host":     property.New(harness.TestMachine.SSHHost),
				"port":     property.New(float64(harness.TestMachine.SSHPort)),
			}),
		}),
	})
	require.NoError(t, err)

	return harness
}

func (harness *ProviderTestHarness) Close() {
	harness.TestMachine.Close()
	harness.Telemetry.Shutdown()
}

func (harness *ProviderTestHarness) AssertCommand(t *testing.T, cmd string) bool {
	session, err := harness.TestMachine.SSHClient.NewSession()
	require.NoError(t, err)
	defer session.Close()

	var stdout strings.Builder
	session.Stdout = &stdout
	var stderr strings.Builder
	session.Stderr = &stderr

	err = session.Run(cmd)
	if !assert.NoError(t, err) {
		t.Logf(
			"command `%s` failed with error %v (%T)\nstdout=%s\nstderr=%s\n",
			cmd,
			err,
			err,
			stdout.String(),
			stderr.String(),
		)
		return false
	}
	return true
}
