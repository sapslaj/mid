package tests

import (
	"strings"
	"testing"

	"github.com/blang/semver"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/integration"
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

	// The expected output for the operation. If ExpectedOutput is nil, no check
	// will be made.
	ExpectedOutput *property.Map

	// A function called on the output of this operation.
	Hook func(t *testing.T, inputs property.Map, output property.Map)

	// If the test should expect the operation to signal an error.
	ExpectFailure bool

	// If CheckFailures is non-nil, expect the check step to fail with the
	// provided output.
	CheckFailures []p.CheckFailure

	// The expected diff for this operation
	ExpectedDiff *p.DiffResponse

	// Command to run to assert test success
	AssertCommand string

	// Command to run before running the operation
	AssertBeforeCommand string

	// Command to run after the dry run but before the real operation
	AssertAfterDryRunCommand string

	// Command to run between delete and create operations in a replace-update
	// operation
	AssertInMiddleOfReplaceCommand string

	// Run a "refresh" (read) before the operation. Ignored on create.
	Refresh bool

	// If the test should expect the "refresh" to fail.
	ExpectRefreshFailure bool

	// A function called on the read outputs
	RefreshHook func(t *testing.T, res p.ReadResponse, err error)

	// Command to run after refresh (if enabled)
	AssertAfterRefreshCommand string
}

type LifeCycleTest struct {
	// Resource token
	Resource string
	// Create operation
	Create Operation
	// Update operations, in order
	Updates []Operation
	// Command to run before the final delete operation
	AssertBeforeDeleteCommand string
	// Command to run after the final delete operation
	AssertDeleteCommand string
}

func (l LifeCycleTest) Run(t *testing.T, harness *ProviderTestHarness) {
	t.Helper()
	urn := MakeURN(l.Resource)

	runCreate := func(op Operation) (p.CreateResponse, bool) {
		t.Log("running create")

		if op.AssertBeforeCommand != "" {
			t.Logf("running before create command %q", op.AssertBeforeCommand)
			if !harness.AssertCommand(t, op.AssertBeforeCommand) {
				t.Log("before create command failed")
				return p.CreateResponse{}, false
			}
		}

		// Here we do the create and the initial setup
		t.Log("running check")
		checkResponse, err := harness.Server.Check(p.CheckRequest{
			Urn:    urn,
			State:  property.Map{},
			Inputs: op.Inputs,
		})
		if !assert.NoError(t, err, "resource check errored") {
			return p.CreateResponse{}, false
		}

		if len(op.CheckFailures) > 0 || len(checkResponse.Failures) > 0 {
			t.Log("checking check failures")
			assert.ElementsMatch(
				t,
				op.CheckFailures,
				checkResponse.Failures,
				"check failures mismatch on create",
			)
			return p.CreateResponse{}, false
		}

		t.Log("dry-run create request")
		_, err = harness.Server.Create(p.CreateRequest{
			Urn:        urn,
			Properties: checkResponse.Inputs,
			DryRun:     true,
		})
		if err != nil {
			t.Logf("got preview create failure: %v", err)
			// We allow the failure from ExpectFailure to hit at either the preview or the Create.
			if op.ExpectFailure {
				t.Log("got expected failure")
			} else {
				t.Fail()
				t.Log("got unexpected failure")
			}
			return p.CreateResponse{}, false
		}

		if op.AssertAfterDryRunCommand != "" {
			t.Logf("running after dry-run create command %q", op.AssertAfterDryRunCommand)
			if !harness.AssertCommand(t, op.AssertAfterDryRunCommand) {
				t.Log("after dry-run create command failed")
				return p.CreateResponse{}, false
			}
		}

		t.Log("create request")
		createResponse, err := harness.Server.Create(p.CreateRequest{
			Urn:        urn,
			Properties: checkResponse.Inputs,
		})
		if op.ExpectFailure {
			assert.Error(t, err, "expected an error on create")
			return p.CreateResponse{}, false
		}
		if !assert.NoError(t, err, "failed to run the create") {
			return p.CreateResponse{}, false
		}

		if op.Hook != nil {
			t.Log("running operation hook")
			op.Hook(t, checkResponse.Inputs, createResponse.Properties)
		}

		if op.ExpectedOutput != nil {
			t.Log("checking for expected output")
			assert.EqualValues(t, *op.ExpectedOutput, createResponse.Properties, "create outputs")
		}

		return createResponse, true
	}

	createResponse, keepGoing := runCreate(l.Create)
	if !keepGoing {
		t.Log("create operation signaled the test to stop")
		return
	}

	if l.Create.AssertCommand != "" {
		t.Logf("running after create command %q", l.Create.AssertCommand)
		if !harness.AssertCommand(t, l.Create.AssertCommand) {
			return
		}
	}

	id := createResponse.ID
	olds := createResponse.Properties

	for i, update := range l.Updates {
		t.Logf("running update %d", i)

		if update.AssertBeforeCommand != "" {
			t.Logf("running before update command %q", update.AssertBeforeCommand)
			if !harness.AssertCommand(t, update.AssertBeforeCommand) {
				t.Log("before update command failed")
				return
			}
		}

		if update.Refresh {
			t.Logf("running refresh %d", i)
			readResponse, err := harness.Server.Read(p.ReadRequest{
				ID:         id,
				Urn:        urn,
				Properties: olds,
				Inputs:     update.Inputs,
			})

			if update.RefreshHook != nil {
				t.Log("running refresh hook")
				update.RefreshHook(t, readResponse, err)
			}

			if update.ExpectRefreshFailure {
				assert.Error(t, err, "expected an error on refresh")
				continue
			}
			if !assert.NoError(t, err, "failed to run the refresh") {
				return
			}

			olds = readResponse.Properties

			if update.AssertAfterRefreshCommand != "" {
				t.Logf("running after create command %q", update.AssertAfterRefreshCommand)
				if !harness.AssertCommand(t, update.AssertAfterRefreshCommand) {
					return
				}
			}
		}

		// Perform the check
		t.Log("running check")
		check, err := harness.Server.Check(p.CheckRequest{
			Urn:    urn,
			State:  olds,
			Inputs: update.Inputs,
		})
		if !assert.NoErrorf(t, err, "check returned an error on update %d", i) {
			return
		}

		if len(update.CheckFailures) > 0 || len(check.Failures) > 0 {
			t.Log("checking check failures")
			assert.ElementsMatchf(
				t,
				update.CheckFailures,
				check.Failures,
				"check failures mismatch on update %d",
				i,
			)
			continue
		}

		t.Log("running diff")
		diff, err := harness.Server.Diff(p.DiffRequest{
			ID:     id,
			Urn:    urn,
			State:  olds,
			Inputs: check.Inputs,
		})
		if !assert.NoErrorf(t, err, "diff failed on update %d", i) {
			return
		}

		if update.ExpectedDiff != nil {
			t.Log("checking diff")
			if !assert.EqualValues(t, *update.ExpectedDiff, diff) {
				return
			}
		}

		if !diff.HasChanges {
			t.Log("no changes, continuing to next operation")
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
			t.Log("detected resource requires a delete and replace")

			runDelete := func() {
				t.Log("running replace-delete")
				err = harness.Server.Delete(p.DeleteRequest{
					ID:         id,
					Urn:        urn,
					Properties: olds,
				})
				assert.NoError(t, err, "failed to delete the resource")
			}

			if diff.DeleteBeforeReplace {
				t.Log("running delete before create")

				runDelete()

				if update.AssertInMiddleOfReplaceCommand != "" {
					t.Logf("running in-middle-of-replace command %q", update.AssertInMiddleOfReplaceCommand)
					if !harness.AssertCommand(t, update.AssertInMiddleOfReplaceCommand) {
						return
					}
				}

				result, keepGoing := runCreate(update)
				if !keepGoing {
					t.Log("create signaled the test to continue to next operation")
					continue
				}

				id = result.ID
				olds = result.Properties
			} else {
				t.Log("running create before delete")
				result, keepGoing := runCreate(update)
				if !keepGoing {
					t.Log("create signaled the test to continue to next operation")
					continue
				}

				if update.AssertInMiddleOfReplaceCommand != "" {
					t.Logf("running in-middle-of-replace command %q", update.AssertInMiddleOfReplaceCommand)
					if !harness.AssertCommand(t, update.AssertInMiddleOfReplaceCommand) {
						return
					}
				}

				runDelete()

				// Set the new block
				id = result.ID
				olds = result.Properties
			}
		} else {
			// Now perform the preview
			t.Log("dry-run update request")
			_, err = harness.Server.Update(p.UpdateRequest{
				ID:     id,
				Urn:    urn,
				State:  olds,
				Inputs: check.Inputs,
				DryRun: true,
			})
			if err != nil {
				t.Logf("got preview update failure: %v", err)
				if update.ExpectFailure {
					t.Log("got expected failure")
					continue
				} else {
					t.Fail()
					t.Log("got unexpected failure")
					return
				}
			}

			if update.AssertAfterDryRunCommand != "" {
				t.Logf("running after dry-run update command %q", update.AssertAfterDryRunCommand)
				if !harness.AssertCommand(t, update.AssertAfterDryRunCommand) {
					t.Log("after dry-run update command failed")
					return
				}
			}

			t.Log("update request")
			result, err := harness.Server.Update(p.UpdateRequest{
				ID:     id,
				Urn:    urn,
				State:  olds,
				Inputs: check.Inputs,
			})
			if update.ExpectFailure {
				if assert.Errorf(t, err, "expected an error on update %d", i) {
					continue
				} else {
					return
				}
			}
			if !assert.NoError(t, err, "failed to update the resource") {
				return
			}

			if update.Hook != nil {
				t.Log("running operation hook")
				update.Hook(t, check.Inputs, result.Properties)
			}

			if update.ExpectedOutput != nil {
				t.Log("checking for expected output")
				assert.EqualValues(t, *update.ExpectedOutput, result.Properties, "expected output on update %d", i)
			}

			olds = result.Properties
		}

		if update.AssertCommand != "" {
			t.Logf("running after update command %q", update.AssertCommand)
			if !harness.AssertCommand(t, update.AssertCommand) {
				return
			}
		}
	}

	if l.AssertBeforeDeleteCommand != "" {
		t.Logf("running before delete command %q", l.AssertBeforeDeleteCommand)
		harness.AssertCommand(t, l.AssertBeforeDeleteCommand)
	}

	t.Log("running delete")
	err := harness.Server.Delete(p.DeleteRequest{
		ID:         id,
		Urn:        urn,
		Properties: olds,
	})
	assert.NoError(t, err, "failed to delete the resource")

	if l.AssertDeleteCommand != "" {
		t.Logf("running after delete command %q", l.AssertDeleteCommand)
		harness.AssertCommand(t, l.AssertDeleteCommand)
	}
}
