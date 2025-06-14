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

type Operation struct {
	// The inputs for the operation
	Inputs property.Map
	// The expected output for the operation. If ExpectedOutput is nil, no check will be made.
	ExpectedOutput *property.Map
	// A function called on the output of this operation.
	Hook func(inputs, output property.Map)
	// If the test should expect the operation to signal an error.
	ExpectFailure bool
	// If CheckFailures is non-nil, expect the check step to fail with the provided output.
	CheckFailures []p.CheckFailure
	// Command to run to assert test success
	AssertCommand string
	// Command to run before running the operation
	AssertBeforeCommand string
}

type LifeCycleTest struct {
	Resource            string
	Create              Operation
	Updates             []Operation
	AssertDeleteCommand string
}

func (l LifeCycleTest) Run(t *testing.T, harness *ProviderTestHarness) {
	t.Helper()
	urn := MakeURN(l.Resource)

	runCreate := func(op Operation) (p.CreateResponse, bool) {
		if op.AssertBeforeCommand != "" && !harness.AssertCommand(t, op.AssertBeforeCommand) {
			return p.CreateResponse{}, false
		}
		// Here we do the create and the initial setup
		checkResponse, err := harness.Server.Check(p.CheckRequest{
			Urn:    urn,
			State:  property.Map{},
			Inputs: op.Inputs,
		})
		assert.NoError(t, err, "resource check errored")
		if len(op.CheckFailures) > 0 || len(checkResponse.Failures) > 0 {
			assert.ElementsMatch(
				t,
				op.CheckFailures,
				checkResponse.Failures,
				"check failures mismatch on create",
			)
			return p.CreateResponse{}, false
		}

		_, err = harness.Server.Create(p.CreateRequest{
			Urn:        urn,
			Properties: checkResponse.Inputs,
			DryRun:     true,
		})
		// We allow the failure from ExpectFailure to hit at either the preview or the Create.
		if op.ExpectFailure && err != nil {
			return p.CreateResponse{}, false
		}
		createResponse, err := harness.Server.Create(p.CreateRequest{
			Urn:        urn,
			Properties: checkResponse.Inputs,
		})
		if op.ExpectFailure {
			assert.Error(t, err, "expected an error on create")
			return p.CreateResponse{}, false
		}
		assert.NoError(t, err, "failed to run the create")
		if err != nil {
			return p.CreateResponse{}, false
		}
		if op.Hook != nil {
			op.Hook(checkResponse.Inputs, createResponse.Properties)
		}
		if op.ExpectedOutput != nil {
			assert.EqualValues(t, *op.ExpectedOutput, createResponse.Properties, "create outputs")
		}
		if op.AssertCommand != "" {
			harness.AssertCommand(t, op.AssertCommand)
		}
		return createResponse, true
	}

	createResponse, keepGoing := runCreate(l.Create)
	if !keepGoing {
		return
	}

	id := createResponse.ID
	olds := createResponse.Properties
	for i, update := range l.Updates {
		if update.AssertBeforeCommand != "" && !harness.AssertCommand(t, update.AssertBeforeCommand) {
			return
		}
		// Perform the check
		check, err := harness.Server.Check(p.CheckRequest{
			Urn:    urn,
			State:  olds,
			Inputs: update.Inputs,
		})

		assert.NoErrorf(t, err, "check returned an error on update %d", i)
		if err != nil {
			return
		}
		if len(update.CheckFailures) > 0 || len(check.Failures) > 0 {
			assert.ElementsMatchf(
				t,
				update.CheckFailures,
				check.Failures,
				"check failures mismatch on update %d",
				i,
			)
			continue
		}

		diff, err := harness.Server.Diff(p.DiffRequest{
			ID:     id,
			Urn:    urn,
			State:  olds,
			Inputs: check.Inputs,
		})
		assert.NoErrorf(t, err, "diff failed on update %d", i)
		if err != nil {
			return
		}
		if !diff.HasChanges {
			// We don't have any changes, so we can just do nothing
			continue
		}
		isDelete := false
		for _, v := range diff.DetailedDiff {
			switch v.Kind {
			case p.AddReplace:
				fallthrough
			case p.DeleteReplace:
				fallthrough
			case p.UpdateReplace:
				isDelete = true
			}
		}
		if isDelete {
			runDelete := func() {
				err = harness.Server.Delete(p.DeleteRequest{
					ID:         id,
					Urn:        urn,
					Properties: olds,
				})
				assert.NoError(t, err, "failed to delete the resource")
			}
			if diff.DeleteBeforeReplace {
				runDelete()
				result, keepGoing := runCreate(update)
				if !keepGoing {
					continue
				}
				id = result.ID
				olds = result.Properties
			} else {
				result, keepGoing := runCreate(update)
				if !keepGoing {
					continue
				}

				runDelete()
				// Set the new block
				id = result.ID
				olds = result.Properties
			}
		} else {

			// Now perform the preview
			_, err = harness.Server.Update(p.UpdateRequest{
				ID:     id,
				Urn:    urn,
				State:  olds,
				Inputs: check.Inputs,
				DryRun: true,
			})

			if update.ExpectFailure && err != nil {
				continue
			}

			result, err := harness.Server.Update(p.UpdateRequest{
				ID:     id,
				Urn:    urn,
				State:  olds,
				Inputs: check.Inputs,
			})
			if !update.ExpectFailure && err != nil {
				assert.NoError(t, err, "failed to update the resource")
				continue
			}
			if update.ExpectFailure {
				assert.Errorf(t, err, "expected failure on update %d", i)
				continue
			}
			if update.Hook != nil {
				update.Hook(check.Inputs, result.Properties)
			}
			if update.ExpectedOutput != nil {
				assert.EqualValues(t, *update.ExpectedOutput, result.Properties, "expected output on update %d", i)
			}
			olds = result.Properties
		}

		if update.AssertCommand != "" && !harness.AssertCommand(t, update.AssertCommand) {
			return
		}
	}
	err := harness.Server.Delete(p.DeleteRequest{
		ID:         id,
		Urn:        urn,
		Properties: olds,
	})
	assert.NoError(t, err, "failed to delete the resource")

	if l.AssertDeleteCommand != "" {
		harness.AssertCommand(t, l.AssertDeleteCommand)
	}
}
