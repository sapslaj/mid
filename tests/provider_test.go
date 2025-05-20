package tests

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/ory/dockertest/v3"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/sapslaj/mid/pkg/telemetry"
	mid "github.com/sapslaj/mid/provider"
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
	Pool          *dockertest.Pool
	Container     *dockertest.Resource
	Client        *ssh.Client
	Provider      integration.Server
	StopTelemetry func()
}

func NewProviderTestHarness(t *testing.T) *ProviderTestHarness {
	t.Helper()

	var err error
	harness := &ProviderTestHarness{}

	t.Log("starting telemetry")
	harness.StopTelemetry = telemetry.StartTelemetry()

	t.Log("starting new dockertest pool")
	harness.Pool, err = dockertest.NewPool("")
	require.NoError(t, err)

	name := "mid-" + strings.ToLower(t.Name())
	t.Logf("running '%s' container", name)
	harness.Container, err = harness.Pool.BuildAndRun(name, "../docker/smoketest/Dockerfile", []string{})
	require.NoError(t, err)

	t.Logf("bound ip: %s", harness.Container.GetBoundIP("22/tcp"))

	port, err := strconv.Atoi(harness.Container.GetPort("22/tcp"))
	require.NoError(t, err)

	addr := fmt.Sprintf(
		"%s:%d",
		harness.Container.GetBoundIP("22/tcp"),
		port,
	)

	for attempt := 1; attempt <= 10; attempt++ {
		t.Logf("(attempt %d/10) connecting to container at address %s over SSH", attempt, addr)
		harness.Client, err = ssh.Dial(
			"tcp",
			addr,
			&ssh.ClientConfig{
				User:            "root",
				Auth:            []ssh.AuthMethod{ssh.Password("hunter2")},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
		)
		if attempt == 10 || err == nil {
			break
		}
		wait := time.Duration(attempt) * 5 * time.Second
		t.Logf("(attempt %d/10) error connecting to container: %v", attempt, err)
		t.Logf("(attempt %d/10) trying again in %s", attempt, wait)
		time.Sleep(wait)
	}
	require.NoError(t, err)

	t.Log("creating and configuring provider")
	harness.Provider = NewProvider()
	err = harness.Provider.Configure(p.ConfigureRequest{
		Args: resource.PropertyMap{
			"connection": resource.NewObjectProperty(resource.PropertyMap{
				"user":     resource.NewStringProperty("root"),
				"password": resource.NewStringProperty("hunter2"),
				"host":     resource.NewStringProperty(harness.Container.GetBoundIP("22/tcp")),
				"port":     resource.NewNumberProperty(float64(port)),
			}),
		},
	})
	require.NoError(t, err)

	return harness
}

func (harness *ProviderTestHarness) Close() {
	if harness.Client != nil {
		harness.Client.Close()
	}
	if harness.Container != nil {
		harness.Pool.Purge(harness.Container)
	}
	harness.StopTelemetry()
}

func (harness *ProviderTestHarness) AssertCommand(t *testing.T, cmd string) bool {
	session, err := harness.Client.NewSession()
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
