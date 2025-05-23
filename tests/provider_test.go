package tests

import (
	"strings"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/pkg/telemetry"
	mid "github.com/sapslaj/mid/provider"
	"github.com/sapslaj/mid/tests/testmachine"
)

func MakeURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(typ), "name")
}

func NewProvider() integration.Server {
	return integration.NewServer(mid.Name, semver.MustParse("1.0.0"), mid.Provider())
}

func Must1[A any](a A, err error) A {
	if err != nil {
		panic(err)
	}
	return a
}

type ProviderTestHarness struct {
	TestMachine   *testmachine.TestMachine
	Provider      integration.Server
	StopTelemetry func()
}

func NewProviderTestHarness(t *testing.T, tmConfig testmachine.Config) *ProviderTestHarness {
	t.Helper()

	var err error
	harness := &ProviderTestHarness{}

	t.Log("starting telemetry")
	harness.StopTelemetry = telemetry.StartTelemetry()

	t.Log("creating new test machine")
	harness.TestMachine, err = testmachine.New(t, tmConfig)
	require.NoError(t, err)

	t.Log("creating and configuring provider")
	harness.Provider = NewProvider()
	err = harness.Provider.Configure(p.ConfigureRequest{
		Args: resource.PropertyMap{
			"connection": resource.NewObjectProperty(resource.PropertyMap{
				"user":     resource.NewStringProperty(harness.TestMachine.SSHUsername),
				"password": resource.NewStringProperty(harness.TestMachine.SSHPassword),
				"host":     resource.NewStringProperty(harness.TestMachine.SSHHost),
				"port":     resource.NewNumberProperty(float64(harness.TestMachine.SSHPort)),
			}),
		},
	})
	require.NoError(t, err)

	return harness
}

func (harness *ProviderTestHarness) Close() {
	harness.TestMachine.Close()
	harness.StopTelemetry()
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
