// Copyright 2016-2024, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"context"
	cryptorand "crypto/rand"
	sha256 "crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"gopkg.in/yaml.v3"

	"github.com/pulumi/pulumi/pkg/v3/engine"
	"github.com/pulumi/pulumi/pkg/v3/operations"
	"github.com/pulumi/pulumi/pkg/v3/resource/stack"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/env"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/config"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/fsutil"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/retry"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/nodejs/npm"
	"github.com/stretchr/testify/assert"
)

const (
	PythonRuntime = "python"
	NodeJSRuntime = "nodejs"
	GoRuntime     = "go"
	DotNetRuntime = "dotnet"
	YAMLRuntime   = "yaml"
	JavaRuntime   = "java"
)

const WindowsOS = "windows"

var ErrTestFailed = errors.New("test failed")

// RuntimeValidationStackInfo contains details related to the stack that runtime validation logic may want to use.
type RuntimeValidationStackInfo struct {
	StackName    tokens.QName
	Deployment   *apitype.DeploymentV3
	RootResource apitype.ResourceV3
	Outputs      map[string]any
	Events       []apitype.EngineEvent
}

// EditDir is an optional edit to apply to the example, as subsequent deployments.
type EditDir struct {
	Dir                    string
	ExtraRuntimeValidation func(t *testing.T, stack RuntimeValidationStackInfo)

	// Additive is true if Dir should be copied *on top* of the test directory.
	// Otherwise Dir *replaces* the test directory, except we keep .pulumi/ and Pulumi.yaml and Pulumi.<stack>.yaml.
	Additive bool

	// ExpectFailure is true if we expect this test to fail.  This is very coarse grained, and will essentially
	// tolerate *any* failure in the program (IDEA: in the future, offer a way to narrow this down more).
	ExpectFailure bool

	// ExpectNoChanges is true if the edit is expected to not propose any changes.
	ExpectNoChanges bool

	// Stdout is the writer to use for all stdout messages.
	Stdout io.Writer
	// Stderr is the writer to use for all stderr messages.
	Stderr io.Writer
	// Verbose may be set to true to print messages as they occur, rather than buffering and showing upon failure.
	Verbose bool

	// Run program directory in query mode.
	QueryMode bool
}

// TestCommandStats is a collection of data related to running a single command during a test.
type TestCommandStats struct {
	// StartTime is the time at which the command was started
	StartTime string `json:"startTime"`
	// EndTime is the time at which the command exited
	EndTime string `json:"endTime"`
	// ElapsedSeconds is the time at which the command exited
	ElapsedSeconds float64 `json:"elapsedSeconds"`
	// StackName is the name of the stack
	StackName string `json:"stackName"`
	// TestId is the unique ID of the test run
	TestID string `json:"testId"`
	// StepName is the command line which was invoked
	StepName string `json:"stepName"`
	// CommandLine is the command line which was invoked
	CommandLine string `json:"commandLine"`
	// TestName is the name of the directory in which the test was executed
	TestName string `json:"testName"`
	// IsError is true if the command failed
	IsError bool `json:"isError"`
	// The Cloud that the test was run against, or empty for local deployments
	CloudURL string `json:"cloudURL"`
}

// TestStatsReporter reports results and metadata from a test run.
type TestStatsReporter interface {
	ReportCommand(stats TestCommandStats)
}

// Environment is used to create environments for use by test programs.
type Environment struct {
	// The name of the environment.
	Name string
	// The definition of the environment.
	Definition map[string]any
}

// ConfigValue is used to provide config values to a test program.
type ConfigValue struct {
	// The config key to pass to `pulumi config`.
	Key string
	// The config value to pass to `pulumi config`.
	Value string
	// Secret indicates that the `--secret` flag should be specified when calling `pulumi config`.
	Secret bool
	// Path indicates that the `--path` flag should be specified when calling `pulumi config`.
	Path bool
}

// ProgramTestOptions provides options for ProgramTest
type ProgramTestOptions struct {
	// Dir is the program directory to test.
	Dir string
	// Array of NPM packages which must be `npm linked` (e.g. {"pulumi", "@pulumi/aws"})
	Dependencies []string
	// Map of package names to versions. The test will use the specified versions of these packages instead of what
	// is declared in `package.json`.
	Overrides map[string]string
	// Automatically use the latest dev version of pulumi SDKs if available.
	InstallDevReleases bool
	// List of environments to create in order.
	CreateEnvironments []Environment
	// List of environments to use.
	Environments []string
	// Map of config keys and values to set (e.g. {"aws:region": "us-east-2"}).
	Config map[string]string
	// Map of secure config keys and values to set (e.g. {"aws:region": "us-east-2"}).
	Secrets map[string]string
	// List of config keys and values to set in order, including Secret and Path options.
	OrderedConfig []ConfigValue
	// SecretsProvider is the optional custom secrets provider to use instead of the default.
	SecretsProvider string
	// EditDirs is an optional list of edits to apply to the example, as subsequent deployments.
	EditDirs []EditDir
	// ExtraRuntimeValidation is an optional callback for additional validation, called before applying edits.
	ExtraRuntimeValidation func(t *testing.T, stack RuntimeValidationStackInfo)
	// RelativeWorkDir is an optional path relative to `Dir` which should be used as working directory during tests.
	RelativeWorkDir string
	// AllowEmptyPreviewChanges is true if we expect that this test's no-op preview may propose changes (e.g.
	// because the test is sensitive to the exact contents of its working directory and those contents change
	// incidentally between the initial update and the empty update).
	AllowEmptyPreviewChanges bool
	// AllowEmptyUpdateChanges is true if we expect that this test's no-op update may perform changes (e.g.
	// because the test is sensitive to the exact contents of its working directory and those contents change
	// incidentally between the initial update and the empty update).
	AllowEmptyUpdateChanges bool
	// ExpectFailure is true if we expect this test to fail.  This is very coarse grained, and will essentially
	// tolerate *any* failure in the program (IDEA: in the future, offer a way to narrow this down more).
	ExpectFailure bool
	// ExpectRefreshChanges may be set to true if a test is expected to have changes yielded by an immediate refresh.
	// This could occur, for example, is a resource's state is constantly changing outside of Pulumi (e.g., timestamps).
	ExpectRefreshChanges bool
	// RetryFailedSteps indicates that failed updates, refreshes, and destroys should be retried after a brief
	// intermission. A maximum of 3 retries will be attempted.
	RetryFailedSteps bool
	// SkipRefresh indicates that the refresh step should be skipped entirely.
	SkipRefresh bool
	// Require a preview after refresh to be a no-op (expect no changes). Has no effect if SkipRefresh is true.
	RequireEmptyPreviewAfterRefresh bool
	// SkipPreview indicates that the preview step should be skipped entirely.
	SkipPreview bool
	// SkipUpdate indicates that the update step should be skipped entirely.
	SkipUpdate bool
	// SkipExportImport skips testing that exporting and importing the stack works properly.
	SkipExportImport bool
	// SkipEmptyPreviewUpdate skips the no-change preview/update that is performed that validates
	// that no changes happen.
	SkipEmptyPreviewUpdate bool
	// SkipStackRemoval indicates that the stack should not be removed. (And so the test's results could be inspected
	// in the Pulumi Service after the test has completed.)
	SkipStackRemoval bool
	// Destroy on cleanup defers stack destruction until the test cleanup step, rather than after
	// program test execution. This is useful for more realistic stack reference testing, allowing one
	// project and stack to be stood up and a second to be run before the first is destroyed.
	//
	// Implies NoParallel because we expect that another caller to ProgramTest will set that
	DestroyOnCleanup bool
	// DestroyExcludeProtected indicates that when the test stack is destroyed,
	// protected resources should be excluded from the destroy operation.
	DestroyExcludeProtected bool
	// Quick implies SkipPreview, SkipExportImport and SkipEmptyPreviewUpdate
	Quick bool
	// RequireService indicates that the test must be run against the Pulumi Service
	RequireService bool
	// PreviewCommandlineFlags specifies flags to add to the `pulumi preview` command line (e.g. "--color=raw")
	PreviewCommandlineFlags []string
	// UpdateCommandlineFlags specifies flags to add to the `pulumi up` command line (e.g. "--color=raw")
	UpdateCommandlineFlags []string
	// QueryCommandlineFlags specifies flags to add to the `pulumi query` command line (e.g. "--color=raw")
	QueryCommandlineFlags []string
	// RunBuild indicates that the build step should be run (e.g. run `npm build` for `nodejs` programs)
	RunBuild bool
	// RunUpdateTest will ensure that updates to the package version can test for spurious diffs
	RunUpdateTest bool
	// DecryptSecretsInOutput will ensure that stack output is passed `--show-secrets` parameter
	// Used in conjunction with ExtraRuntimeValidation
	DecryptSecretsInOutput bool

	// CloudURL is an optional URL to override the default Pulumi Service API (https://api.pulumi-staging.io). The
	// PULUMI_ACCESS_TOKEN environment variable must also be set to a valid access token for the target cloud.
	CloudURL string

	// StackName allows the stack name to be explicitly provided instead of computed from the
	// environment during tests.
	StackName string

	// If non-empty, specifies the value of the `--tracing` flag to pass
	// to Pulumi CLI, which may be a Zipkin endpoint or a
	// `file:./local.trace` style url for AppDash tracing.
	//
	// Template `{command}` syntax will be expanded to the current
	// command name such as `pulumi-stack-rm`. This is useful for
	// file-based tracing since `ProgramTest` performs multiple
	// CLI invocations that can inadvertently overwrite the trace
	// file.
	Tracing string

	// NoParallel will opt the test out of being ran in parallel.
	NoParallel bool

	// PrePulumiCommand specifies a callback that will be executed before each `pulumi` invocation. This callback may
	// optionally return another callback to be invoked after the `pulumi` invocation completes.
	PrePulumiCommand func(verb string) (func(err error) error, error)

	// ReportStats optionally specifies how to report results from the test for external collection.
	ReportStats TestStatsReporter

	// Stdout is the writer to use for all stdout messages.
	Stdout io.Writer
	// Stderr is the writer to use for all stderr messages.
	Stderr io.Writer
	// Verbose may be set to true to print messages as they occur, rather than buffering and showing upon failure.
	Verbose bool

	// DebugLogLevel may be set to anything >0 to enable excessively verbose debug logging from `pulumi`. This
	// is equivalent to `--logflow --logtostderr -v=N`, where N is the value of DebugLogLevel. This may also
	// be enabled by setting the environment variable PULUMI_TEST_DEBUG_LOG_LEVEL.
	DebugLogLevel int
	// DebugUpdates may be set to true to enable debug logging from `pulumi preview`, `pulumi up`, and
	// `pulumi destroy`.  This may also be enabled by setting the environment variable PULUMI_TEST_DEBUG_UPDATES.
	DebugUpdates bool

	// Bin is a location of a `pulumi` executable to be run.  Taken from the $PATH if missing.
	Bin string
	// NpmBin is a location of a `npm` executable to be run.  Taken from the $PATH if missing.
	NpmBin string
	// GoBin is a location of a `go` executable to be run.  Taken from the $PATH if missing.
	GoBin string
	// PythonBin is a location of a `python` executable to be run.  Taken from the $PATH if missing.
	PythonBin string
	// PipenvBin is a location of a `pipenv` executable to run.  Taken from the $PATH if missing.
	PipenvBin string
	// DotNetBin is a location of a `dotnet` executable to be run.  Taken from the $PATH if missing.
	DotNetBin string

	// Additional environment variables to pass for each command we run.
	Env []string

	// Automatically create and use a virtual environment, rather than using the Pipenv tool. This is now the default
	// behavior, so this option no longer has any affect. To go back to the old behavior use the `UsePipenv` option.
	UseAutomaticVirtualEnv bool
	// Use the Pipenv tool to manage the virtual environment.
	UsePipenv bool
	// Use a shared virtual environment for tests based on the contents of the requirements file. Defaults to false.
	UseSharedVirtualEnv *bool
	// Shared venv path when UseSharedVirtualEnv is true. Defaults to $HOME/.pulumi-test-venvs.
	SharedVirtualEnvPath string
	// Refers to the shared venv directory when UseSharedVirtualEnv is true. Otherwise defaults to venv
	VirtualEnvDir string

	// If set, this hook is called after the `pulumi preview` command has completed.
	PreviewCompletedHook func(dir string) error

	// JSONOutput indicates that the `--json` flag should be passed to `up`, `preview`,
	// `refresh` and `destroy` commands.
	JSONOutput bool

	// If set, this hook is called after `pulumi stack export` on the exported file. If `SkipExportImport` is set, this
	// hook is ignored.
	ExportStateValidator func(t *testing.T, stack []byte)

	// If not nil, specifies the logic of preparing a project by
	// ensuring dependencies. If left as nil, runs default
	// preparation logic by dispatching on whether the project
	// uses Node, Python, .NET or Go.
	PrepareProject func(*engine.Projinfo) error

	// If not nil, will be run before the project has been prepared.
	PrePrepareProject func(*engine.Projinfo) error

	// If not nil, will be run after the project has been prepared.
	PostPrepareProject func(*engine.Projinfo) error

	// Array of provider plugin dependencies which come from local packages.
	LocalProviders []LocalDependency

	// The directory to use for PULUMI_HOME. Useful for benchmarks where you want to run a warmup run of `ProgramTest`
	// to download plugins before running the timed run of `ProgramTest`.
	PulumiHomeDir string
}

func (opts *ProgramTestOptions) GetUseSharedVirtualEnv() bool {
	if opts.UseSharedVirtualEnv != nil {
		return *opts.UseSharedVirtualEnv
	}
	return false
}

type LocalDependency struct {
	Package string
	Path    string
}

func (opts *ProgramTestOptions) GetDebugLogLevel() int {
	if opts.DebugLogLevel > 0 {
		return opts.DebugLogLevel
	}
	if du := os.Getenv("PULUMI_TEST_DEBUG_LOG_LEVEL"); du != "" {
		if n, e := strconv.Atoi(du); e != nil {
			panic(e)
		} else if n > 0 {
			return n
		}
	}
	return 0
}

func (opts *ProgramTestOptions) GetDebugUpdates() bool {
	return opts.DebugUpdates || os.Getenv("PULUMI_TEST_DEBUG_UPDATES") != ""
}

// GetStackName returns a stack name to use for this test.
func (opts *ProgramTestOptions) GetStackName() tokens.QName {
	if opts.StackName == "" {
		// Fetch the host and test dir names, cleaned so to contain just [a-zA-Z0-9-_] chars.
		hostname, err := os.Hostname()
		contract.AssertNoErrorf(err, "failure to fetch hostname for stack prefix")
		var host string
		for _, c := range hostname {
			if len(host) >= 10 {
				break
			}
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '_' {
				host += string(c)
			}
		}

		var test string
		for _, c := range filepath.Base(opts.Dir) {
			if len(test) >= 10 {
				break
			}
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '_' {
				test += string(c)
			}
		}

		b := make([]byte, 4)
		_, err = cryptorand.Read(b)
		contract.AssertNoErrorf(err, "failure to generate random stack suffix")

		opts.StackName = strings.ToLower("p-it-" + host + "-" + test + "-" + hex.EncodeToString(b))
	}

	return tokens.QName(opts.StackName)
}

// GetEnvName returns the uniquified name for the given environment. The name is made unique by appending the FNV hash
// of the associated stack's name. This ensures that the name is both unique and deterministic. The name must be
// deterministic because it is computed by both LifeCycleInitialize and TestLifeCycleDestroy.
func (opts *ProgramTestOptions) GetEnvName(name string) string {
	h := fnv.New32()
	_, err := h.Write([]byte(opts.GetStackName()))
	contract.IgnoreError(err)

	suffix := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%v-%v", name, suffix)
}

func (opts *ProgramTestOptions) GetEnvNameWithOwner(name string) string {
	owner := os.Getenv("PULUMI_TEST_OWNER")
	if opts.RequireService && owner != "" {
		return fmt.Sprintf("%v/%v", owner, opts.GetEnvName(name))
	}
	return opts.GetEnvName(name)
}

// Returns the md5 hash of the file at the given path as a string
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	buf := make([]byte, 32*1024)
	hash := sha256.New()
	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, err := hash.Write(buf[:n])
			if err != nil {
				return "", err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	sum := string(hash.Sum(nil))
	return sum, nil
}

// GetStackNameWithOwner gets the name of the stack prepended with an owner, if PULUMI_TEST_OWNER is set.
// We use this in CI to create test stacks in an organization that all developers have access to, for debugging.
func (opts *ProgramTestOptions) GetStackNameWithOwner() tokens.QName {
	owner := os.Getenv("PULUMI_TEST_OWNER")

	if opts.RequireService && owner != "" {
		return tokens.QName(fmt.Sprintf("%s/%s", owner, opts.GetStackName()))
	}

	return opts.GetStackName()
}

// With combines a source set of options with a set of overrides.
func (opts ProgramTestOptions) With(overrides ProgramTestOptions) ProgramTestOptions {
	if overrides.Dir != "" {
		opts.Dir = overrides.Dir
	}
	if overrides.Dependencies != nil {
		opts.Dependencies = overrides.Dependencies
	}
	if overrides.Overrides != nil {
		opts.Overrides = overrides.Overrides
	}
	if overrides.InstallDevReleases {
		opts.InstallDevReleases = overrides.InstallDevReleases
	}
	if len(overrides.CreateEnvironments) != 0 {
		opts.CreateEnvironments = append(opts.CreateEnvironments, overrides.CreateEnvironments...)
	}
	if len(overrides.Environments) != 0 {
		opts.Environments = append(opts.Environments, overrides.Environments...)
	}
	for k, v := range overrides.Config {
		if opts.Config == nil {
			opts.Config = make(map[string]string)
		}
		opts.Config[k] = v
	}
	for k, v := range overrides.Secrets {
		if opts.Secrets == nil {
			opts.Secrets = make(map[string]string)
		}
		opts.Secrets[k] = v
	}
	if overrides.OrderedConfig != nil {
		opts.OrderedConfig = append(opts.OrderedConfig, overrides.OrderedConfig...)
	}
	if overrides.SecretsProvider != "" {
		opts.SecretsProvider = overrides.SecretsProvider
	}
	if overrides.EditDirs != nil {
		opts.EditDirs = overrides.EditDirs
	}
	if overrides.ExtraRuntimeValidation != nil {
		opts.ExtraRuntimeValidation = overrides.ExtraRuntimeValidation
	}
	if overrides.RelativeWorkDir != "" {
		opts.RelativeWorkDir = overrides.RelativeWorkDir
	}
	if overrides.AllowEmptyPreviewChanges {
		opts.AllowEmptyPreviewChanges = overrides.AllowEmptyPreviewChanges
	}
	if overrides.AllowEmptyUpdateChanges {
		opts.AllowEmptyUpdateChanges = overrides.AllowEmptyUpdateChanges
	}
	if overrides.ExpectFailure {
		opts.ExpectFailure = overrides.ExpectFailure
	}
	if overrides.ExpectRefreshChanges {
		opts.ExpectRefreshChanges = overrides.ExpectRefreshChanges
	}
	if overrides.RetryFailedSteps {
		opts.RetryFailedSteps = overrides.RetryFailedSteps
	}
	if overrides.SkipRefresh {
		opts.SkipRefresh = overrides.SkipRefresh
	}
	if overrides.RequireEmptyPreviewAfterRefresh {
		opts.RequireEmptyPreviewAfterRefresh = overrides.RequireEmptyPreviewAfterRefresh
	}
	if overrides.SkipPreview {
		opts.SkipPreview = overrides.SkipPreview
	}
	if overrides.SkipUpdate {
		opts.SkipUpdate = overrides.SkipUpdate
	}
	if overrides.SkipExportImport {
		opts.SkipExportImport = overrides.SkipExportImport
	}
	if overrides.SkipEmptyPreviewUpdate {
		opts.SkipEmptyPreviewUpdate = overrides.SkipEmptyPreviewUpdate
	}
	if overrides.SkipStackRemoval {
		opts.SkipStackRemoval = overrides.SkipStackRemoval
	}
	if overrides.DestroyOnCleanup {
		opts.DestroyOnCleanup = overrides.DestroyOnCleanup
	}
	if overrides.DestroyExcludeProtected {
		opts.DestroyExcludeProtected = overrides.DestroyExcludeProtected
	}
	if overrides.Quick {
		opts.Quick = overrides.Quick
	}
	if overrides.RequireService {
		opts.RequireService = overrides.RequireService
	}
	if overrides.PreviewCommandlineFlags != nil {
		opts.PreviewCommandlineFlags = append(opts.PreviewCommandlineFlags, overrides.PreviewCommandlineFlags...)
	}
	if overrides.UpdateCommandlineFlags != nil {
		opts.UpdateCommandlineFlags = append(opts.UpdateCommandlineFlags, overrides.UpdateCommandlineFlags...)
	}
	if overrides.QueryCommandlineFlags != nil {
		opts.QueryCommandlineFlags = append(opts.QueryCommandlineFlags, overrides.QueryCommandlineFlags...)
	}
	if overrides.RunBuild {
		opts.RunBuild = overrides.RunBuild
	}
	if overrides.RunUpdateTest {
		opts.RunUpdateTest = overrides.RunUpdateTest
	}
	if overrides.DecryptSecretsInOutput {
		opts.DecryptSecretsInOutput = overrides.DecryptSecretsInOutput
	}
	if overrides.CloudURL != "" {
		opts.CloudURL = overrides.CloudURL
	}
	if overrides.StackName != "" {
		opts.StackName = overrides.StackName
	}
	if overrides.Tracing != "" {
		opts.Tracing = overrides.Tracing
	}
	if overrides.NoParallel {
		opts.NoParallel = overrides.NoParallel
	}
	if overrides.PrePulumiCommand != nil {
		opts.PrePulumiCommand = overrides.PrePulumiCommand
	}
	if overrides.ReportStats != nil {
		opts.ReportStats = overrides.ReportStats
	}
	if overrides.Stdout != nil {
		opts.Stdout = overrides.Stdout
	}
	if overrides.Stderr != nil {
		opts.Stderr = overrides.Stderr
	}
	if overrides.Verbose {
		opts.Verbose = overrides.Verbose
	}
	if overrides.DebugLogLevel != 0 {
		opts.DebugLogLevel = overrides.DebugLogLevel
	}
	if overrides.DebugUpdates {
		opts.DebugUpdates = overrides.DebugUpdates
	}
	if overrides.Bin != "" {
		opts.Bin = overrides.Bin
	}
	if overrides.NpmBin != "" {
		opts.NpmBin = overrides.NpmBin
	}
	if overrides.GoBin != "" {
		opts.GoBin = overrides.GoBin
	}
	if overrides.PipenvBin != "" {
		opts.PipenvBin = overrides.PipenvBin
	}
	if overrides.DotNetBin != "" {
		opts.DotNetBin = overrides.DotNetBin
	}
	if overrides.Env != nil {
		opts.Env = append(opts.Env, overrides.Env...)
	}
	if overrides.UseAutomaticVirtualEnv {
		opts.UseAutomaticVirtualEnv = overrides.UseAutomaticVirtualEnv
	}
	if overrides.UsePipenv {
		opts.UsePipenv = overrides.UsePipenv
	}
	if overrides.UseSharedVirtualEnv != nil {
		opts.UseSharedVirtualEnv = overrides.UseSharedVirtualEnv
	}
	if overrides.SharedVirtualEnvPath != "" {
		opts.SharedVirtualEnvPath = overrides.SharedVirtualEnvPath
	}
	if overrides.PreviewCompletedHook != nil {
		opts.PreviewCompletedHook = overrides.PreviewCompletedHook
	}
	if overrides.JSONOutput {
		opts.JSONOutput = overrides.JSONOutput
	}
	if overrides.ExportStateValidator != nil {
		opts.ExportStateValidator = overrides.ExportStateValidator
	}
	if overrides.PrepareProject != nil {
		opts.PrepareProject = overrides.PrepareProject
	}
	if overrides.PrePrepareProject != nil {
		opts.PrePrepareProject = overrides.PrePrepareProject
	}
	if overrides.PostPrepareProject != nil {
		opts.PostPrepareProject = overrides.PostPrepareProject
	}
	if overrides.LocalProviders != nil {
		opts.LocalProviders = append(opts.LocalProviders, overrides.LocalProviders...)
	}
	if overrides.PulumiHomeDir != "" {
		opts.PulumiHomeDir = overrides.PulumiHomeDir
	}
	return opts
}

type RegexFlag struct {
	Re *regexp.Regexp
}

func (rf *RegexFlag) String() string {
	if rf.Re == nil {
		return ""
	}
	return rf.Re.String()
}

func (rf *RegexFlag) Set(v string) error {
	r, err := regexp.Compile(v)
	if err != nil {
		return err
	}
	rf.Re = r
	return nil
}

var (
	DirectoryMatcher RegexFlag
	ListDirs         bool
	PipMutex         *fsutil.FileMutex
)

func init() {
	flag.Var(&DirectoryMatcher, "dirs", "optional list of regexes to use to select integration tests to run")
	flag.BoolVar(&ListDirs, "list-dirs", false, "list available integration tests without running them")

	mutexPath := filepath.Join(os.TempDir(), "pip-mutex.lock")
	PipMutex = fsutil.NewFileMutex(mutexPath)
}

// GetLogs retrieves the logs for a given stack in a particular region making the query provided.
//
// [provider] should be one of "aws" or "azure"
func GetLogs(
	t *testing.T,
	provider, region string,
	stackInfo RuntimeValidationStackInfo,
	query operations.LogQuery,
) *[]operations.LogEntry {
	snap, err := stack.DeserializeDeploymentV3(
		context.Background(),
		*stackInfo.Deployment,
		stack.DefaultSecretsProvider)
	assert.NoError(t, err)

	tree := operations.NewResourceTree(snap.Resources)
	if !assert.NotNil(t, tree) {
		return nil
	}

	cfg := map[config.Key]string{
		config.MustMakeKey(provider, "region"): region,
	}
	ops := tree.OperationsProvider(cfg)

	// Validate logs from example
	logs, err := ops.GetLogs(query)
	if !assert.NoError(t, err) {
		return nil
	}

	return logs
}

func PrepareProgram(t *testing.T, opts *ProgramTestOptions) {
	// If we're just listing tests, simply print this test's directory.
	if ListDirs {
		fmt.Printf("%s\n", opts.Dir)
	}

	// If we have a matcher, ensure that this test matches its pattern.
	if DirectoryMatcher.Re != nil && !DirectoryMatcher.Re.Match([]byte(opts.Dir)) {
		t.Skipf("Skipping: '%v' does not match '%v'", opts.Dir, DirectoryMatcher.Re)
	}

	// Disable stack backups for tests to avoid filling up ~/.pulumi/backups with unnecessary
	// backups of test stacks.
	disableCheckpointBackups := env.DIYBackendDisableCheckpointBackups.Var().Name()
	opts.Env = append(opts.Env, disableCheckpointBackups+"=1")

	// We want tests to default into being ran in parallel, hence the odd double negative.
	if !opts.NoParallel && !opts.DestroyOnCleanup {
		t.Parallel()
	}

	if os.Getenv("PULUMI_TEST_USE_SERVICE") == "true" {
		opts.RequireService = true
	}
	if opts.RequireService {
		// This token is set in CI jobs, so this escape hatch is here to enable a smooth local dev
		// experience, i.e.: running "make" and not seeing many failures due to a missing token.
		if os.Getenv("PULUMI_ACCESS_TOKEN") == "" {
			t.Skipf("Skipping: PULUMI_ACCESS_TOKEN is not set")
		}
	} else if opts.CloudURL == "" {
		opts.CloudURL = MakeTempBackend(t)
	}

	// If the test panics, recover and log instead of letting the panic escape the test. Even though *this* test will
	// have run deferred functions and cleaned up, if the panic reaches toplevel it will kill the process and prevent
	// other tests running in parallel from cleaning up.
	defer func() {
		if failure := recover(); failure != nil {
			t.Errorf("panic testing %v: %v", opts.Dir, failure)
		}
	}()

	// Set up some default values for sending test reports and tracing data. We use environment varaiables to
	// control these globally and set reasonable values for our own use in CI.
	if opts.ReportStats == nil {
		if v := os.Getenv("PULUMI_TEST_REPORT_CONFIG"); v != "" {
			splits := strings.Split(v, ":")
			if len(splits) != 3 {
				t.Errorf("report config should be set to a value of the form: <aws-region>:<bucket-name>:<keyPrefix>")
			}

			opts.ReportStats = NewS3Reporter(splits[0], splits[1], splits[2])
		}
	}

	if opts.Tracing == "" {
		opts.Tracing = os.Getenv("PULUMI_TEST_TRACE_ENDPOINT")
	}

	if opts.UseSharedVirtualEnv == nil {
		if sharedVenv := os.Getenv("PULUMI_TEST_PYTHON_SHARED_VENV"); sharedVenv != "" {
			useSharedVenvBool := sharedVenv == "true"
			opts.UseSharedVirtualEnv = &useSharedVenvBool
		}
	}

	if opts.VirtualEnvDir == "" && !opts.GetUseSharedVirtualEnv() {
		opts.VirtualEnvDir = "venv"
	}

	if opts.SharedVirtualEnvPath == "" {
		opts.SharedVirtualEnvPath = filepath.Join(os.Getenv("HOME"), ".pulumi-test-venvs")
		if sharedVenvPath := os.Getenv("PULUMI_TEST_PYTHON_SHARED_VENV_PATH"); sharedVenvPath != "" {
			opts.SharedVirtualEnvPath = sharedVenvPath
		}
	}

	if opts.Quick {
		opts.SkipPreview = true
		opts.SkipExportImport = true
		opts.SkipEmptyPreviewUpdate = true
	}
}

// ProgramTest runs a lifecycle of Pulumi commands in a program working
// directory, using the `pulumi` and `npm` binaries available on PATH.  It
// essentially executes the following workflow:
//
//	npm install
//	npm link <each opts.Depencies>
//	(+) npm run build
//	pulumi init
//	(*) pulumi login
//	pulumi stack init integrationtesting
//	pulumi config set <each opts.Config>
//	pulumi config set --secret <each opts.Secrets>
//	pulumi preview
//	pulumi up
//	pulumi stack export --file stack.json
//	pulumi stack import --file stack.json
//	pulumi preview (expected to be empty)
//	pulumi up (expected to be empty)
//	pulumi destroy --yes
//	pulumi stack rm --yes integrationtesting
//
//	(*) Only if PULUMI_ACCESS_TOKEN is set.
//	(+) Only if `opts.RunBuild` is true.
//
// All commands must return success return codes for the test to succeed, unless ExpectFailure is true.
func ProgramTest(t *testing.T, opts *ProgramTestOptions) {
	pt := ProgramTestManualLifeCycle(t, opts)
	err := pt.TestLifeCycleInitAndDestroy()
	if !errors.Is(err, ErrTestFailed) {
		assert.NoError(t, err)
	}
}

// ProgramTestManualLifeCycle returns a ProgramTester than must be manually controlled in terms of its lifecycle
func ProgramTestManualLifeCycle(t *testing.T, opts *ProgramTestOptions) *ProgramTester {
	PrepareProgram(t, opts)
	pt := NewProgramTester(t, opts)
	return pt
}

// ProgramTester contains state associated with running a single test pass.
type ProgramTester struct {
	T              *testing.T          // the Go tester for this run.
	Opts           *ProgramTestOptions // options that control this test run.
	Bin            string              // the `pulumi` binary we are using.
	NpmBin         string              // the `npm` binary we are using
	GoBin          string              // the `go` binary we are using.
	PythonBin      string              // the `python` binary we are using.
	PipenvBin      string              // The `pipenv` binary we are using.
	DotNetBin      string              // the `dotnet` binary we are using.
	UpdateEventLog string              // The path to the engine event log for `pulumi up` in this test.
	MaxStepTries   int                 // The maximum number of times to retry a failed pulumi step.
	Tmpdir         string              // the temporary directory we use for our test environment
	Projdir        string              // the project directory we use for this run
	TestFinished   bool                // whether or not the test if finished
	PulumiHome     string              // The directory PULUMI_HOME will be set to
}

func NewProgramTester(t *testing.T, opts *ProgramTestOptions) *ProgramTester {
	stackName := opts.GetStackName()
	maxStepTries := 1
	if opts.RetryFailedSteps {
		maxStepTries = 3
	}
	var home string
	if opts.PulumiHomeDir != "" {
		home = opts.PulumiHomeDir
	} else {
		home = t.TempDir()
	}
	return &ProgramTester{
		T:              t,
		Opts:           opts,
		UpdateEventLog: filepath.Join(os.TempDir(), string(stackName)+"-events.json"),
		MaxStepTries:   maxStepTries,
		PulumiHome:     home,
	}
}

// MakeTempBackend creates a temporary backend directory which will clean up on test exit.
func MakeTempBackend(t *testing.T) string {
	tempDir := t.TempDir()
	return "file://" + filepath.ToSlash(tempDir)
}

func (pt *ProgramTester) GetTmpDir() string {
	return pt.Tmpdir
}

func (pt *ProgramTester) GetBin() (string, error) {
	return GetCmdBin(&pt.Bin, "pulumi", pt.Opts.Bin)
}

func (pt *ProgramTester) GetNpmBin() (string, error) {
	return GetCmdBin(&pt.NpmBin, "npm", pt.Opts.NpmBin)
}

func (pt *ProgramTester) GetGoBin() (string, error) {
	return GetCmdBin(&pt.GoBin, "go", pt.Opts.GoBin)
}

// GetPythonBin returns a path to the currently-installed `python` binary, or an error if it could not be found.
func (pt *ProgramTester) GetPythonBin() (string, error) {
	if pt.PythonBin == "" {
		pt.PythonBin = pt.Opts.PythonBin
		if pt.Opts.PythonBin == "" {
			var err error
			// Look for `python3` by default, but fallback to `python` if not found, except on Windows
			// where we look for these in the reverse order because the default python.org Windows
			// installation does not include a `python3` binary, and the existence of a `python3.exe`
			// symlink to `python.exe` on some systems does not work correctly with the Python `venv`
			// module.
			pythonCmds := []string{"python3", "python"}
			if runtime.GOOS == WindowsOS {
				pythonCmds = []string{"python", "python3"}
			}
			for _, bin := range pythonCmds {
				pt.PythonBin, err = exec.LookPath(bin)
				// Break on the first cmd we find on the path (if any).
				if err == nil {
					break
				}
			}
			if err != nil {
				return "", fmt.Errorf("Expected to find one of %q on $PATH: %w", pythonCmds, err)
			}
		}
	}
	return pt.PythonBin, nil
}

// GetPipenvBin returns a path to the currently-installed Pipenv tool, or an error if the tool could not be found.
func (pt *ProgramTester) GetPipenvBin() (string, error) {
	return GetCmdBin(&pt.PipenvBin, "pipenv", pt.Opts.PipenvBin)
}

func (pt *ProgramTester) GetDotNetBin() (string, error) {
	return GetCmdBin(&pt.DotNetBin, "dotnet", pt.Opts.DotNetBin)
}

func (pt *ProgramTester) PulumiCmd(name string, args []string) ([]string, error) {
	bin, err := pt.GetBin()
	if err != nil {
		return nil, err
	}
	cmd := []string{bin}
	if du := pt.Opts.GetDebugLogLevel(); du > 0 {
		cmd = append(cmd, "--logflow", "--logtostderr", "-v="+strconv.Itoa(du))
	}
	cmd = append(cmd, args...)
	if tracing := pt.Opts.Tracing; tracing != "" {
		cmd = append(cmd, "--tracing", strings.ReplaceAll(tracing, "{command}", name))
	}
	return cmd, nil
}

func (pt *ProgramTester) NpmCmd(args []string) ([]string, error) {
	bin, err := pt.GetNpmBin()
	if err != nil {
		return nil, err
	}
	result := []string{bin}
	result = append(result, args...)
	return WithOptionalNpmFlags(result), nil
}

func (pt *ProgramTester) PythonCmd(args []string) ([]string, error) {
	bin, err := pt.GetPythonBin()
	if err != nil {
		return nil, err
	}

	cmd := []string{bin}
	return append(cmd, args...), nil
}

func (pt *ProgramTester) PipenvCmd(args []string) ([]string, error) {
	bin, err := pt.GetPipenvBin()
	if err != nil {
		return nil, err
	}

	cmd := []string{bin}
	return append(cmd, args...), nil
}

func (pt *ProgramTester) RunCommand(name string, args []string, wd string) error {
	return RunCommandPulumiHome(pt.T, name, args, wd, pt.Opts, pt.PulumiHome)
}

// RunPulumiCommand runs a Pulumi command in the project directory.
// For example:
//
//	pt.RunPulumiCommand("preview", "--stack", "dev")
func (pt *ProgramTester) RunPulumiCommand(name string, args ...string) error {
	// pt.runPulumiCommand uses 'name' for logging only.
	// We want it to be part of the actual command.
	args = append([]string{name}, args...)
	return pt.RunPulumiCommandWD(name, args, pt.Projdir, false /* expectFailure */)
}

func (pt *ProgramTester) RunPulumiCommandWD(name string, args []string, wd string, expectFailure bool) error {
	cmd, err := pt.PulumiCmd(name, args)
	if err != nil {
		return err
	}

	var postFn func(error) error
	if pt.Opts.PrePulumiCommand != nil {
		postFn, err = pt.Opts.PrePulumiCommand(args[0])
		if err != nil {
			return err
		}
	}

	isUpdate := args[0] == "preview" || args[0] == "up" || args[0] == "destroy" || args[0] == "refresh"

	// If we're doing a preview or an update and this project is a Python project, we need to run
	// the command in the context of the virtual environment that Pipenv created in order to pick up
	// the correct version of Python.  We also need to do this for destroy and refresh so that
	// dynamic providers are run in the right virtual environment.
	// This is only necessary when not using automatic virtual environment support.
	if pt.Opts.UsePipenv && isUpdate {
		projinfo, err := pt.GetProjinfo(wd)
		if err != nil {
			return nil
		}

		if projinfo.Proj.Runtime.Name() == "python" {
			pipenvBin, err := pt.GetPipenvBin()
			if err != nil {
				return err
			}

			// "pipenv run" activates the current virtual environment and runs the remainder of the arguments as if it
			// were a command.
			cmd = append([]string{pipenvBin, "run"}, cmd...)
		}
	}

	_, _, err = retry.Until(context.Background(), retry.Acceptor{
		Accept: func(try int, nextRetryTime time.Duration) (bool, interface{}, error) {
			runerr := pt.RunCommand(name, cmd, wd)
			if runerr == nil {
				return true, nil, nil
			} else if _, ok := runerr.(*exec.ExitError); ok && isUpdate && !expectFailure {
				// the update command failed, let's try again, assuming we haven't failed a few times.
				if try+1 >= pt.MaxStepTries {
					return false, nil, fmt.Errorf("%v did not succeed after %v tries", cmd, try+1)
				}

				pt.T.Logf("%v failed: %v; retrying...", cmd, runerr)
				return false, nil, nil
			}

			// some other error, fail
			return false, nil, runerr
		},
	})
	if postFn != nil {
		if postErr := postFn(err); postErr != nil {
			return multierror.Append(err, postErr)
		}
	}
	return err
}

func (pt *ProgramTester) RunNpmCommand(name string, args []string, wd string) error {
	cmd, err := pt.NpmCmd(args)
	if err != nil {
		return err
	}

	_, _, err = retry.Until(context.Background(), retry.Acceptor{
		Accept: func(try int, nextRetryTime time.Duration) (bool, interface{}, error) {
			runerr := pt.RunCommand(name, cmd, wd)
			if runerr == nil {
				return true, nil, nil
			} else if _, ok := runerr.(*exec.ExitError); ok {
				// npm failed, let's try again, assuming we haven't failed a few times.
				if try+1 >= 3 {
					return false, nil, fmt.Errorf("%v did not complete after %v tries", cmd, try+1)
				}

				return false, nil, nil
			}

			// someother error, fail
			return false, nil, runerr
		},
	})
	return err
}

func (pt *ProgramTester) RunPythonCommand(name string, args []string, wd string) error {
	cmd, err := pt.PythonCmd(args)
	if err != nil {
		return err
	}

	return pt.RunCommand(name, cmd, wd)
}

func (pt *ProgramTester) RunVirtualEnvCommand(name string, args []string, wd string) error {
	// When installing with `pip install -e`, a PKG-INFO file is created. If two packages are being installed
	// this way simultaneously (which happens often, when running tests), both installations will be writing the
	// same file simultaneously. If one process catches "PKG-INFO" in a half-written state, the one process that
	// observed the torn write will fail to install the package.
	//
	// To avoid this problem, we use pipMutex to explicitly serialize installation operations. Doing so avoids
	// the problem of multiple processes stomping on the same files in the source tree. Note that pipMutex is a
	// file mutex, so this strategy works even if the go test runner chooses to split up text execution across
	// multiple processes. (Furthermore, each test gets an instance of ProgramTester and thus the mutex, so we'd
	// need to be sharing the mutex globally in each test process if we weren't using the file system to lock.)
	if name == "virtualenv-pip-install-package" {
		if err := PipMutex.Lock(); err != nil {
			panic(err)
		}

		if pt.Opts.Verbose {
			pt.T.Log("acquired pip install lock")
			defer pt.T.Log("released pip install lock")
		}
		defer func() {
			if err := PipMutex.Unlock(); err != nil {
				panic(err)
			}
		}()
	}

	virtualenvBinPath, err := GetVirtualenvBinPath(wd, args[0], pt)
	if err != nil {
		return err
	}

	cmd := append([]string{virtualenvBinPath}, args[1:]...)
	return pt.RunCommand(name, cmd, wd)
}

func (pt *ProgramTester) RunPipenvCommand(name string, args []string, wd string) error {
	// Pipenv uses setuptools to install and uninstall packages. Setuptools has an installation mode called "develop"
	// that we use to install the package being tested, since it is 1) lightweight and 2) not doing so has its own set
	// of annoying problems.
	//
	// Setuptools develop does three things:
	//   1. It invokes the "egg_info" command in the target package,
	//   2. It creates a special `.egg-link` sentinel file in the current site-packages folder, pointing to the package
	//      being installed's path on disk
	//   3. It updates easy-install.pth in site-packages so that pip understand that this package has been installed.
	//
	// Steps 2 and 3 operate entirely within the context of a virtualenv. The state that they mutate is fully contained
	// within the current virtualenv. However, step 1 operates in the context of the package's source tree. Egg info
	// is responsible for producing a minimal "egg" for a particular package, and its largest responsibility is creating
	// a PKG-INFO file for a package. PKG-INFO contains, among other things, the version of the package being installed.
	//
	// If two packages are being installed in "develop" mode simultaneously (which happens often, when running tests),
	// both installations will run "egg_info" on the source tree and both processes will be writing the same files
	// simultaneously. If one process catches "PKG-INFO" in a half-written state, the one process that observed the
	// torn write will fail to install the package (setuptools crashes).
	//
	// To avoid this problem, we use pipMutex to explicitly serialize installation operations. Doing so avoids the
	// problem of multiple processes stomping on the same files in the source tree. Note that pipMutex is a file
	// mutex, so this strategy works even if the go test runner chooses to split up text execution across multiple
	// processes. (Furthermore, each test gets an instance of ProgramTester and thus the mutex, so we'd need to be
	// sharing the mutex globally in each test process if we weren't using the file system to lock.)
	if name == "pipenv-install-package" {
		if err := PipMutex.Lock(); err != nil {
			panic(err)
		}

		if pt.Opts.Verbose {
			pt.T.Log("acquired pip install lock")
			defer pt.T.Log("released pip install lock")
		}
		defer func() {
			if err := PipMutex.Unlock(); err != nil {
				panic(err)
			}
		}()
	}

	cmd, err := pt.PipenvCmd(args)
	if err != nil {
		return err
	}

	return pt.RunCommand(name, cmd, wd)
}

// TestLifeCyclePrepare prepares a test by creating a temporary directory
func (pt *ProgramTester) TestLifeCyclePrepare() error {
	tmpdir, projdir, err := pt.CopyTestToTemporaryDirectory()
	pt.Tmpdir = tmpdir
	pt.Projdir = projdir
	return err
}

func (pt *ProgramTester) CheckTestFailure() error {
	if pt.T.Failed() {
		pt.T.Logf("Canceling further steps due to test failure")
		return ErrTestFailed
	}
	return nil
}

// TestCleanUp cleans up the temporary directory that a test used
func (pt *ProgramTester) TestCleanUp() {
	testFinished := pt.TestFinished
	if pt.Tmpdir != "" {
		if !testFinished || pt.T.Failed() {
			// Test aborted or failed. Maybe copy to "failed tests" directory.
			failedTestsDir := os.Getenv("PULUMI_FAILED_TESTS_DIR")
			if failedTestsDir != "" {
				dest := filepath.Join(failedTestsDir, pt.T.Name()+UniqueSuffix())
				contract.IgnoreError(fsutil.CopyFile(dest, pt.Tmpdir, nil))
			}
		} else {
			contract.IgnoreError(os.RemoveAll(pt.Tmpdir))
		}
	} else {
		// When tmpdir is empty, we ran "in tree", which means we wrote output
		// to the "command-output" folder in the projdir, and we should clean
		// it up if the test passed
		if testFinished && !pt.T.Failed() {
			contract.IgnoreError(os.RemoveAll(filepath.Join(pt.Projdir, CommandOutputFolderName)))
		}
	}

	// Clean up the temporary PULUMI_HOME directory we created. This is necessary to reclaim the disk space of the
	// plugins that were downloaded during the test. We only created this if `opts.PulumiHomeDir` is empty, otherwise we
	// will have used the provided directory and should leave it alone.
	if pt.Opts.PulumiHomeDir == "" {
		contract.IgnoreError(os.RemoveAll(pt.PulumiHome))
	}
}

// TestLifeCycleInitAndDestroy executes the test and cleans up
func (pt *ProgramTester) TestLifeCycleInitAndDestroy() error {
	err := pt.TestLifeCyclePrepare()
	if err != nil {
		return fmt.Errorf("copying test to temp dir %s: %w", pt.Tmpdir, err)
	}

	pt.TestFinished = false
	if pt.Opts.DestroyOnCleanup {
		pt.T.Cleanup(pt.TestCleanUp)
	} else {
		defer pt.TestCleanUp()
	}

	err = pt.TestLifeCycleInitialize()
	if err != nil {
		return fmt.Errorf("initializing test project: %w", err)
	}

	destroyStack := func() {
		destroyErr := pt.TestLifeCycleDestroy()
		assert.NoError(pt.T, destroyErr)
	}
	if pt.Opts.DestroyOnCleanup {
		// Allow other tests to refer to this stack until the test is complete.
		pt.T.Cleanup(destroyStack)
	} else {
		// Ensure that before we exit, we attempt to destroy and remove the stack.
		defer destroyStack()
	}

	if err = pt.TestPreviewUpdateAndEdits(); err != nil {
		return fmt.Errorf("running test preview, update, and edits: %w", err)
	}

	if pt.Opts.RunUpdateTest {
		err = UpgradeProjectDeps(pt.Projdir, pt)
		if err != nil {
			return fmt.Errorf("upgrading project dependencies: %w", err)
		}

		if err = pt.TestPreviewUpdateAndEdits(); err != nil {
			return fmt.Errorf("running test preview, update, and edits (updateTest): %w", err)
		}
	}

	pt.TestFinished = true
	return nil
}

func UpgradeProjectDeps(projectDir string, pt *ProgramTester) error {
	projInfo, err := pt.GetProjinfo(projectDir)
	if err != nil {
		return fmt.Errorf("getting project info: %w", err)
	}

	switch rt := projInfo.Proj.Runtime.Name(); rt {
	case NodeJSRuntime:
		if err = pt.NpmLinkPackageDeps(projectDir); err != nil {
			return err
		}
	case PythonRuntime:
		if err = pt.InstallPipPackageDeps(projectDir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized project runtime: %s", rt)
	}

	return nil
}

// TestLifeCycleInitialize initializes the project directory and stack along with any configuration
func (pt *ProgramTester) TestLifeCycleInitialize() error {
	dir := pt.Projdir
	stackName := pt.Opts.GetStackName()

	// Set the default target Pulumi API if not overridden in options.
	if pt.Opts.CloudURL == "" {
		pulumiAPI := os.Getenv("PULUMI_API")
		if pulumiAPI != "" {
			pt.Opts.CloudURL = pulumiAPI
		}
	}

	// Ensure all links are present, the stack is created, and all configs are applied.
	pt.T.Logf("Initializing project (dir %s; stack %s)", dir, stackName)

	// Login as needed.
	stackInitName := string(pt.Opts.GetStackNameWithOwner())

	if os.Getenv("PULUMI_ACCESS_TOKEN") == "" && pt.Opts.CloudURL == "" {
		fmt.Printf("Using existing logged in user for tests.  Set PULUMI_ACCESS_TOKEN and/or PULUMI_API to override.\n")
	} else {
		// Set PulumiCredentialsPathEnvVar to our CWD, so we use credentials specific to just this
		// test.
		pt.Opts.Env = append(pt.Opts.Env, fmt.Sprintf("%s=%s", workspace.PulumiCredentialsPathEnvVar, dir))

		loginArgs := []string{"login"}
		loginArgs = AddFlagIfNonNil(loginArgs, "--cloud-url", pt.Opts.CloudURL)

		// If this is a local OR cloud login, then don't attach the owner to the stack-name.
		if pt.Opts.CloudURL != "" {
			stackInitName = string(pt.Opts.GetStackName())
		}

		if err := pt.RunPulumiCommandWD("pulumi-login", loginArgs, dir, false); err != nil {
			return err
		}
	}

	// Stack init
	stackInitArgs := []string{"stack", "init", stackInitName}
	if pt.Opts.SecretsProvider != "" {
		stackInitArgs = append(stackInitArgs, "--secrets-provider", pt.Opts.SecretsProvider)
	}
	if err := pt.RunPulumiCommandWD("pulumi-stack-init", stackInitArgs, dir, false); err != nil {
		return err
	}

	if len(pt.Opts.Config)+len(pt.Opts.Secrets) > 0 {
		setAllArgs := []string{"config", "set-all"}

		for key, value := range pt.Opts.Config {
			setAllArgs = append(setAllArgs, "--plaintext", fmt.Sprintf("%s=%s", key, value))
		}
		for key, value := range pt.Opts.Secrets {
			setAllArgs = append(setAllArgs, "--secret", fmt.Sprintf("%s=%s", key, value))
		}

		if err := pt.RunPulumiCommandWD("pulumi-config", setAllArgs, dir, false); err != nil {
			return err
		}
	}

	for _, cv := range pt.Opts.OrderedConfig {
		configArgs := []string{"config", "set", cv.Key, cv.Value}
		if cv.Secret {
			configArgs = append(configArgs, "--secret")
		}
		if cv.Path {
			configArgs = append(configArgs, "--path")
		}
		if err := pt.RunPulumiCommandWD("pulumi-config", configArgs, dir, false); err != nil {
			return err
		}
	}

	// Environments
	for _, env := range pt.Opts.CreateEnvironments {
		name := pt.Opts.GetEnvNameWithOwner(env.Name)

		envFile, err := func() (string, error) {
			temp, err := os.CreateTemp(pt.T.TempDir(), fmt.Sprintf("pulumi-env-%v-*", env.Name))
			if err != nil {
				return "", err
			}
			defer contract.IgnoreClose(temp)

			enc := yaml.NewEncoder(temp)
			enc.SetIndent(2)
			if err = enc.Encode(env.Definition); err != nil {
				return "", err
			}
			return temp.Name(), nil
		}()
		if err != nil {
			return err
		}

		initArgs := []string{"env", "init", name, "-f", envFile}
		if err := pt.RunPulumiCommandWD("pulumi-env-init", initArgs, dir, false); err != nil {
			return err
		}
	}

	if len(pt.Opts.Environments) != 0 {
		envs := make([]string, len(pt.Opts.Environments))
		for i, e := range pt.Opts.Environments {
			envs[i] = pt.Opts.GetEnvName(e)
		}

		stackFile := filepath.Join(dir, fmt.Sprintf("Pulumi.%v.yaml", stackName))
		bytes, err := os.ReadFile(stackFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		var stack workspace.ProjectStack
		if err := yaml.Unmarshal(bytes, &stack); err != nil {
			return err
		}
		stack.Environment = workspace.NewEnvironment(envs)

		bytes, err = yaml.Marshal(stack)
		if err != nil {
			return err
		}

		if err = os.WriteFile(stackFile, bytes, 0o600); err != nil {
			return err
		}
	}

	return nil
}

// TestLifeCycleDestroy destroys a stack and removes it
func (pt *ProgramTester) TestLifeCycleDestroy() error {
	if pt.Projdir != "" {
		// Destroy and remove the stack.
		pt.T.Log("Destroying stack")
		destroy := []string{"destroy", "--non-interactive", "--yes", "--skip-preview"}
		if pt.Opts.GetDebugUpdates() {
			destroy = append(destroy, "-d")
		}
		if pt.Opts.JSONOutput {
			destroy = append(destroy, "--json")
		}
		if pt.Opts.DestroyExcludeProtected {
			destroy = append(destroy, "--exclude-protected")
		}
		if err := pt.RunPulumiCommandWD("pulumi-destroy", destroy, pt.Projdir, false); err != nil {
			return err
		}

		if pt.T.Failed() {
			pt.T.Logf("Test failed, retaining stack '%s'", pt.Opts.GetStackNameWithOwner())
			return nil
		}

		if !pt.Opts.SkipStackRemoval {
			err := pt.RunPulumiCommandWD("pulumi-stack-rm", []string{"stack", "rm", "--yes"}, pt.Projdir, false)
			if err != nil {
				return err
			}
		}

		for _, env := range pt.Opts.CreateEnvironments {
			name := pt.Opts.GetEnvNameWithOwner(env.Name)
			err := pt.RunPulumiCommandWD("pulumi-env-rm", []string{"env", "rm", "--yes", name}, pt.Projdir, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// TestPreviewUpdateAndEdits runs the preview, update, and any relevant edits
func (pt *ProgramTester) TestPreviewUpdateAndEdits() error {
	dir := pt.Projdir
	// Now preview and update the real changes.
	pt.T.Log("Performing primary preview and update")
	initErr := pt.PreviewAndUpdate(dir, "initial", pt.Opts.ExpectFailure, false, false)

	// If the initial preview/update failed, just exit without trying the rest (but make sure to destroy).
	if initErr != nil {
		return fmt.Errorf("initial failure: %w", initErr)
	}

	// Perform an empty preview and update; nothing is expected to happen here.
	if !pt.Opts.SkipExportImport {
		pt.T.Log("Roundtripping checkpoint via stack export and stack import")

		if err := pt.ExportImport(dir); err != nil {
			return fmt.Errorf("empty preview + update: %w", err)
		}
	}

	if !pt.Opts.SkipEmptyPreviewUpdate {
		msg := ""
		if !pt.Opts.AllowEmptyUpdateChanges {
			msg = "(no changes expected)"
		}
		pt.T.Logf("Performing empty preview and update%s", msg)
		if err := pt.PreviewAndUpdate(dir, "empty", pt.Opts.ExpectFailure,
			!pt.Opts.AllowEmptyPreviewChanges, !pt.Opts.AllowEmptyUpdateChanges); err != nil {
			return fmt.Errorf("empty preview: %w", err)
		}
	}

	// Run additional validation provided by the test options, passing in the checkpoint info.
	if err := pt.PerformExtraRuntimeValidation(pt.Opts.ExtraRuntimeValidation, dir); err != nil {
		return err
	}

	if !pt.Opts.SkipRefresh {
		// Perform a refresh and ensure it doesn't yield changes.
		refresh := []string{"refresh", "--non-interactive", "--yes", "--skip-preview"}
		if pt.Opts.GetDebugUpdates() {
			refresh = append(refresh, "-d")
		}
		if pt.Opts.JSONOutput {
			refresh = append(refresh, "--json")
		}
		if !pt.Opts.ExpectRefreshChanges {
			refresh = append(refresh, "--expect-no-changes")
		}
		if err := pt.RunPulumiCommandWD("pulumi-refresh", refresh, dir, false); err != nil {
			return err
		}

		// Perform another preview and expect no changes in it.
		if pt.Opts.RequireEmptyPreviewAfterRefresh {
			preview := []string{"preview", "--non-interactive", "--expect-no-changes"}
			if pt.Opts.GetDebugUpdates() {
				preview = append(preview, "-d")
			}
			if pt.Opts.JSONOutput {
				preview = append(preview, "--json")
			}
			if pt.Opts.PreviewCommandlineFlags != nil {
				preview = append(preview, pt.Opts.PreviewCommandlineFlags...)
			}
			if err := pt.RunPulumiCommandWD("pulumi-preview-after-refresh", preview, dir, false); err != nil {
				return err
			}
		}
	}

	// If there are any edits, apply them and run a preview and update for each one.
	return pt.TestEdits(dir)
}

func (pt *ProgramTester) ExportImport(dir string) error {
	exportCmd := []string{"stack", "export", "--file", "stack.json"}
	importCmd := []string{"stack", "import", "--file", "stack.json"}

	defer func() {
		contract.IgnoreError(os.Remove(filepath.Join(dir, "stack.json")))
	}()

	if err := pt.RunPulumiCommandWD("pulumi-stack-export", exportCmd, dir, false); err != nil {
		return err
	}

	if f := pt.Opts.ExportStateValidator; f != nil {
		bytes, err := os.ReadFile(filepath.Join(dir, "stack.json"))
		if err != nil {
			pt.T.Logf("Failed to read stack.json: %s", err)
			return err
		}
		pt.T.Logf("Calling ExportStateValidator")
		f(pt.T, bytes)

		if err := pt.CheckTestFailure(); err != nil {
			return err
		}
	}

	return pt.RunPulumiCommandWD("pulumi-stack-import", importCmd, dir, false)
}

// PreviewAndUpdate runs pulumi preview followed by pulumi up
func (pt *ProgramTester) PreviewAndUpdate(dir string, name string, shouldFail, expectNopPreview,
	expectNopUpdate bool,
) error {
	preview := []string{"preview", "--non-interactive", "--diff"}
	update := []string{"up", "--non-interactive", "--yes", "--skip-preview", "--event-log", pt.UpdateEventLog}
	if pt.Opts.GetDebugUpdates() {
		preview = append(preview, "-d")
		update = append(update, "-d")
	}
	if pt.Opts.JSONOutput {
		preview = append(preview, "--json")
		update = append(update, "--json")
	}
	if expectNopPreview {
		preview = append(preview, "--expect-no-changes")
	}
	if expectNopUpdate {
		update = append(update, "--expect-no-changes")
	}
	if pt.Opts.PreviewCommandlineFlags != nil {
		preview = append(preview, pt.Opts.PreviewCommandlineFlags...)
	}
	if pt.Opts.UpdateCommandlineFlags != nil {
		update = append(update, pt.Opts.UpdateCommandlineFlags...)
	}

	// If not in quick mode, run an explicit preview.
	if !pt.Opts.SkipPreview {
		if err := pt.RunPulumiCommandWD("pulumi-preview-"+name, preview, dir, shouldFail); err != nil {
			if shouldFail {
				pt.T.Log("Permitting failure (ExpectFailure=true for this preview)")
				return nil
			}
			return err
		}
		if pt.Opts.PreviewCompletedHook != nil {
			if err := pt.Opts.PreviewCompletedHook(dir); err != nil {
				return err
			}
		}
	}

	// Now run an update.
	if !pt.Opts.SkipUpdate {
		if err := pt.RunPulumiCommandWD("pulumi-update-"+name, update, dir, shouldFail); err != nil {
			if shouldFail {
				pt.T.Log("Permitting failure (ExpectFailure=true for this update)")
				return nil
			}
			return err
		}
	}

	// If we expected a failure, but none occurred, return an error.
	if shouldFail {
		return errors.New("expected this step to fail, but it succeeded")
	}

	return nil
}

func (pt *ProgramTester) Query(dir string, name string, shouldFail bool) error {
	query := []string{"query", "--non-interactive"}
	if pt.Opts.GetDebugUpdates() {
		query = append(query, "-d")
	}
	if pt.Opts.QueryCommandlineFlags != nil {
		query = append(query, pt.Opts.QueryCommandlineFlags...)
	}

	// Now run a query.
	if err := pt.RunPulumiCommandWD("pulumi-query-"+name, query, dir, shouldFail); err != nil {
		if shouldFail {
			pt.T.Log("Permitting failure (ExpectFailure=true for this update)")
			return nil
		}
		return err
	}

	// If we expected a failure, but none occurred, return an error.
	if shouldFail {
		return errors.New("expected this step to fail, but it succeeded")
	}

	return nil
}

func (pt *ProgramTester) TestEdits(dir string) error {
	for i, edit := range pt.Opts.EditDirs {
		var err error
		if err = pt.TestEdit(dir, i, edit); err != nil {
			return err
		}
	}
	return nil
}

func (pt *ProgramTester) TestEdit(dir string, i int, edit EditDir) error {
	pt.T.Logf("Applying edit '%v' and rerunning preview and update", edit.Dir)

	if edit.Additive {
		// Just copy new files into dir
		if err := fsutil.CopyFile(dir, edit.Dir, nil); err != nil {
			return fmt.Errorf("Couldn't copy %v into %v: %w", edit.Dir, dir, err)
		}
	} else {
		// Create a new temporary directory
		newDir, err := os.MkdirTemp("", pt.Opts.StackName+"-")
		if err != nil {
			return fmt.Errorf("Couldn't create new temporary directory: %w", err)
		}

		// Delete whichever copy of the test is unused when we return
		dirToDelete := newDir
		defer func() {
			contract.IgnoreError(os.RemoveAll(dirToDelete))
		}()

		// Copy everything except Pulumi.yaml, Pulumi.<stack-name>.yaml, and .pulumi from source into new directory
		exclusions := make(map[string]bool)
		projectYaml := workspace.ProjectFile + ".yaml"
		configYaml := workspace.ProjectFile + "." + pt.Opts.StackName + ".yaml"
		exclusions[workspace.BookkeepingDir] = true
		exclusions[projectYaml] = true
		exclusions[configYaml] = true

		if err := fsutil.CopyFile(newDir, edit.Dir, exclusions); err != nil {
			return fmt.Errorf("Couldn't copy %v into %v: %w", edit.Dir, newDir, err)
		}

		// Copy Pulumi.yaml, Pulumi.<stack-name>.yaml, and .pulumi from old directory to new directory
		oldProjectYaml := filepath.Join(dir, projectYaml)
		newProjectYaml := filepath.Join(newDir, projectYaml)

		oldConfigYaml := filepath.Join(dir, configYaml)
		newConfigYaml := filepath.Join(newDir, configYaml)

		oldProjectDir := filepath.Join(dir, workspace.BookkeepingDir)
		newProjectDir := filepath.Join(newDir, workspace.BookkeepingDir)

		if err := fsutil.CopyFile(newProjectYaml, oldProjectYaml, nil); err != nil {
			return fmt.Errorf("Couldn't copy Pulumi.yaml: %w", err)
		}

		// Copy the config file over if it exists.
		//
		// Pulumi is not required to write a config file if there is no config, so
		// it might not.
		if _, err := os.Stat(oldConfigYaml); !os.IsNotExist(err) {
			if err := fsutil.CopyFile(newConfigYaml, oldConfigYaml, nil); err != nil {
				return fmt.Errorf("Couldn't copy Pulumi.%s.yaml: %w", pt.Opts.StackName, err)
			}
		}

		// Likewise, pulumi is not required to write a book-keeping (.pulumi) file.
		if _, err := os.Stat(oldProjectDir); !os.IsNotExist(err) {
			if err := fsutil.CopyFile(newProjectDir, oldProjectDir, nil); err != nil {
				return fmt.Errorf("Couldn't copy .pulumi: %w", err)
			}
		}

		// Finally, replace our current temp directory with the new one.
		dirOld := dir + ".old"
		if err := os.Rename(dir, dirOld); err != nil {
			return fmt.Errorf("Couldn't rename %v to %v: %w", dir, dirOld, err)
		}

		// There's a brief window here where the old temp dir name could be taken from us.

		if err := os.Rename(newDir, dir); err != nil {
			return fmt.Errorf("Couldn't rename %v to %v: %w", newDir, dir, err)
		}

		// Keep dir, delete oldDir
		dirToDelete = dirOld
	}

	err := pt.PrepareProjectDir(dir)
	if err != nil {
		return fmt.Errorf("Couldn't prepare project in %v: %w", dir, err)
	}

	oldStdOut := pt.Opts.Stdout
	oldStderr := pt.Opts.Stderr
	oldVerbose := pt.Opts.Verbose
	if edit.Stdout != nil {
		pt.Opts.Stdout = edit.Stdout
	}
	if edit.Stderr != nil {
		pt.Opts.Stderr = edit.Stderr
	}
	if edit.Verbose {
		pt.Opts.Verbose = true
	}

	defer func() {
		pt.Opts.Stdout = oldStdOut
		pt.Opts.Stderr = oldStderr
		pt.Opts.Verbose = oldVerbose
	}()

	if !edit.QueryMode {
		if err = pt.PreviewAndUpdate(dir, fmt.Sprintf("edit-%d", i),
			edit.ExpectFailure, edit.ExpectNoChanges, edit.ExpectNoChanges); err != nil {
			return err
		}
	} else {
		if err = pt.Query(dir, fmt.Sprintf("query-%d", i), edit.ExpectFailure); err != nil {
			return err
		}
	}
	return pt.PerformExtraRuntimeValidation(edit.ExtraRuntimeValidation, dir)
}

func (pt *ProgramTester) PerformExtraRuntimeValidation(
	extraRuntimeValidation func(t *testing.T, stack RuntimeValidationStackInfo), dir string,
) error {
	if extraRuntimeValidation == nil {
		return nil
	}

	stackName := pt.Opts.GetStackName()

	// Create a temporary file name for the stack export
	tempDir, err := os.MkdirTemp("", string(stackName))
	if err != nil {
		return err
	}
	fileName := filepath.Join(tempDir, "stack.json")

	// Invoke `pulumi stack export`
	// There are situations where we want to get access to the secrets in the validation
	// this will allow us to get access to them as part of running ExtraRuntimeValidation
	var pulumiCommand []string
	if pt.Opts.DecryptSecretsInOutput {
		pulumiCommand = append(pulumiCommand, "stack", "export", "--show-secrets", "--file", fileName)
	} else {
		pulumiCommand = append(pulumiCommand, "stack", "export", "--file", fileName)
	}
	if err = pt.RunPulumiCommandWD("pulumi-export",
		pulumiCommand, dir, false); err != nil {
		return fmt.Errorf("expected to export stack to file: %s: %w", fileName, err)
	}

	// Open the exported JSON file
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("expected to be able to open file with stack exports: %s: %w", fileName, err)
	}
	defer func() {
		contract.IgnoreClose(f)
		contract.IgnoreError(os.RemoveAll(tempDir))
	}()

	// Unmarshal the Deployment
	var untypedDeployment apitype.UntypedDeployment
	if err = json.NewDecoder(f).Decode(&untypedDeployment); err != nil {
		return err
	}
	var deployment apitype.DeploymentV3
	if err = json.Unmarshal(untypedDeployment.Deployment, &deployment); err != nil {
		return err
	}

	// Get the root resource and outputs from the deployment
	var rootResource apitype.ResourceV3
	var outputs map[string]interface{}
	for _, res := range deployment.Resources {
		if res.Type == resource.RootStackType && res.Parent == "" {
			rootResource = res
			outputs = res.Outputs
		}
	}

	events, err := pt.readUpdateEventLog()
	if err != nil {
		return err
	}

	// Populate stack info object with all of this data to pass to the validation function
	stackInfo := RuntimeValidationStackInfo{
		StackName:    pt.Opts.GetStackName(),
		Deployment:   &deployment,
		RootResource: rootResource,
		Outputs:      outputs,
		Events:       events,
	}

	pt.T.Log("Performing extra runtime validation.")
	extraRuntimeValidation(pt.T, stackInfo)
	pt.T.Log("Extra runtime validation complete.")

	return pt.CheckTestFailure()
}

func (pt *ProgramTester) readUpdateEventLog() ([]apitype.EngineEvent, error) {
	events := []apitype.EngineEvent{}
	eventsFile, err := os.Open(pt.UpdateEventLog)
	if err != nil {
		if os.IsNotExist(err) {
			return events, nil
		}
		return events, fmt.Errorf("expected to be able to open event log file %s: %w",
			pt.UpdateEventLog, err)
	}

	defer contract.IgnoreClose(eventsFile)

	decoder := json.NewDecoder(eventsFile)
	for {
		var event apitype.EngineEvent
		if err = decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return events, fmt.Errorf("failed decoding engine event from log file %s: %w",
				pt.UpdateEventLog, err)
		}
		events = append(events, event)
	}

	return events, nil
}

// CopyTestToTemporaryDirectory creates a temporary directory to run the test in and copies the test to it.
func (pt *ProgramTester) CopyTestToTemporaryDirectory() (string, string, error) {
	// Get the source dir and project info.
	sourceDir := pt.Opts.Dir
	projSourceDir := sourceDir
	if wd := pt.Opts.RelativeWorkDir; wd != "" {
		projSourceDir = filepath.Join(projSourceDir, wd)
	}
	projinfo, err := pt.GetProjinfo(projSourceDir)
	if err != nil {
		return "", "", fmt.Errorf("could not get project info from source: %w", err)
	}

	if pt.Opts.Stdout == nil {
		pt.Opts.Stdout = os.Stdout
	}
	if pt.Opts.Stderr == nil {
		pt.Opts.Stderr = os.Stderr
	}

	pt.T.Logf("sample: %v", sourceDir)
	bin, err := pt.GetBin()
	if err != nil {
		return "", "", err
	}
	pt.T.Logf("pulumi: %v\n", bin)

	stackName := string(pt.Opts.GetStackName())

	// For most projects, we will copy to a temporary directory.  For Go projects, however, we must create
	// a folder structure that adheres to GOPATH requirements
	var tmpdir, projdir string
	if projinfo.Proj.Runtime.Name() == "go" {
		targetDir, err := CreateTemporaryGoFolder("stackName")
		if err != nil {
			return "", "", fmt.Errorf("Couldn't create temporary directory: %w", err)
		}
		tmpdir = targetDir
		projdir = targetDir
	} else {
		targetDir, tempErr := os.MkdirTemp("", stackName+"-")
		if tempErr != nil {
			return "", "", fmt.Errorf("Couldn't create temporary directory: %w", tempErr)
		}
		tmpdir = targetDir
		projdir = targetDir
	}
	if wd := pt.Opts.RelativeWorkDir; wd != "" {
		projdir = filepath.Join(projdir, wd)
	}
	// Copy the source project.
	if copyErr := fsutil.CopyFile(tmpdir, sourceDir, nil); copyErr != nil {
		return "", "", copyErr
	}
	// Reload the projinfo before making mutating changes (workspace.LoadProject caches the in-memory Project by path)
	projinfo, err = pt.GetProjinfo(projdir)
	if err != nil {
		return "", "", fmt.Errorf("could not get project info: %w", err)
	}

	// Add dynamic plugin paths from ProgramTester
	if (projinfo.Proj.Plugins == nil || projinfo.Proj.Plugins.Providers == nil) && pt.Opts.LocalProviders != nil {
		projinfo.Proj.Plugins = &workspace.Plugins{
			Providers: make([]workspace.PluginOptions, 0),
		}
	}

	if pt.Opts.LocalProviders != nil {
		for _, provider := range pt.Opts.LocalProviders {
			// LocalProviders are relative to the working directory when running tests, NOT relative to the
			// Pulumi.yaml. This is a bit odd, but makes it easier to construct the required paths in each
			// test.
			absPath, err := filepath.Abs(provider.Path)
			if err != nil {
				return "", "", fmt.Errorf("could not get absolute path for plugin %s: %w", provider.Path, err)
			}

			projinfo.Proj.Plugins.Providers = append(projinfo.Proj.Plugins.Providers, workspace.PluginOptions{
				Name: provider.Package,
				Path: absPath,
			})
		}
	}

	// Absolute path of the source directory, for fixupPath to use below
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", "", fmt.Errorf("could not get absolute path for source directory %s: %w", sourceDir, err)
	}

	// Return a fixed up path if it's relative to sourceDir but not beneath it, else just returns the input
	fixupPath := func(path string) (string, error) {
		if filepath.IsAbs(path) {
			return path, nil
		}
		absPlugin := filepath.Join(absSource, path)
		if !strings.HasPrefix(absPlugin, absSource+string(filepath.Separator)) {
			return absPlugin, nil
		}
		return path, nil
	}

	if projinfo.Proj.Plugins != nil {
		optionSets := [][]workspace.PluginOptions{
			projinfo.Proj.Plugins.Providers,
			projinfo.Proj.Plugins.Languages,
			projinfo.Proj.Plugins.Analyzers,
		}
		for _, options := range optionSets {
			for i, opt := range options {
				path, err := fixupPath(opt.Path)
				if err != nil {
					return "", "", fmt.Errorf("could not get fixed path for plugin %s: %w", opt.Path, err)
				}
				options[i].Path = path
			}
		}
	}
	projfile := filepath.Join(projdir, workspace.ProjectFile+".yaml")
	bytes, err := yaml.Marshal(projinfo.Proj)
	if err != nil {
		return "", "", fmt.Errorf("error marshalling project %q: %w", projfile, err)
	}

	if err := os.WriteFile(projfile, bytes, 0o600); err != nil {
		return "", "", fmt.Errorf("error writing project: %w", err)
	}

	if pt.Opts.PrePrepareProject != nil {
		err = pt.Opts.PrePrepareProject(projinfo)
		if err != nil {
			return "", "", fmt.Errorf("Failed to pre-prepare %v: %w", projdir, err)
		}
	}

	err = pt.PrepareProject(projinfo)
	if err != nil {
		return "", "", fmt.Errorf("Failed to prepare %v: %w", projdir, err)
	}

	if pt.Opts.PostPrepareProject != nil {
		err = pt.Opts.PostPrepareProject(projinfo)
		if err != nil {
			return "", "", fmt.Errorf("Failed to post-prepare %v: %w", projdir, err)
		}
	}

	// TODO[pulumi/pulumi#5455]: Dynamic providers fail to load when used from multi-lang components.
	// Until that's been fixed, this environment variable can be set by a test, which results in
	// a package.json being emitted in the project directory and `npm install && npm link @pulumi/pulumi`
	// being run.
	// When the underlying issue has been fixed, the use of this environment variable should be removed.
	var npmLinkPulumi bool
	for _, env := range pt.Opts.Env {
		if env == "PULUMI_TEST_YARN_LINK_PULUMI=true" || env == "PULUMI_TEST_NPM_LINK_PULUMI=true" {
			npmLinkPulumi = true
			break
		}
	}
	if npmLinkPulumi {
		const packageJSON = `{
			"name": "test",
			"peerDependencies": {
				"@pulumi/pulumi": "latest"
			}
		}`
		if err := os.WriteFile(filepath.Join(projdir, "package.json"), []byte(packageJSON), 0o600); err != nil {
			return "", "", err
		}
		if err := pt.RunNpmCommand("npm-link", []string{"link", "@pulumi/pulumi"}, projdir); err != nil {
			return "", "", err
		}
		if err = pt.RunNpmCommand("npm-install", []string{"install"}, projdir); err != nil {
			return "", "", err
		}
	}

	pt.T.Logf("projdir: %v", projdir)
	return tmpdir, projdir, nil
}

func (pt *ProgramTester) GetProjinfo(projectDir string) (*engine.Projinfo, error) {
	// Load up the package so we know things like what language the project is.
	projfile := filepath.Join(projectDir, workspace.ProjectFile+".yaml")
	proj, err := workspace.LoadProject(projfile)
	if err != nil {
		return nil, err
	}
	return &engine.Projinfo{Proj: proj, Root: projectDir}, nil
}

// PrepareProject runs setup necessary to get the project ready for `pulumi` commands.
func (pt *ProgramTester) PrepareProject(projinfo *engine.Projinfo) error {
	if pt.Opts.PrepareProject != nil {
		return pt.Opts.PrepareProject(projinfo)
	}
	return pt.DefaultPrepareProject(projinfo)
}

// PrepareProjectDir runs setup necessary to get the project ready for `pulumi` commands.
func (pt *ProgramTester) PrepareProjectDir(projectDir string) error {
	projinfo, err := pt.GetProjinfo(projectDir)
	if err != nil {
		return err
	}
	return pt.PrepareProject(projinfo)
}

// PrepareNodeJSProject runs setup necessary to get a Node.js project ready for `pulumi` commands.
func (pt *ProgramTester) PrepareNodeJSProject(projinfo *engine.Projinfo) error {
	// Get the correct pwd to run Npm in.
	cwd, _, err := projinfo.GetPwdMain()
	if err != nil {
		return err
	}

	workspaceRoot, err := npm.FindWorkspaceRoot(cwd)
	if err != nil {
		if !errors.Is(err, npm.ErrNotInWorkspace) {
			return err
		}
		// Not in a workspace, don't updated cwd.
	} else {
		pt.T.Logf("detected yarn/npm workspace root at %s", workspaceRoot)
		cwd = workspaceRoot
	}

	// If dev versions were requested, we need to update the
	// package.json to use them.  Note that Overrides take
	// priority over installing dev versions.
	if pt.Opts.InstallDevReleases {
		err := pt.RunNpmCommand("npm-install", []string{"install", "--save", "@pulumi/pulumi@dev"}, cwd)
		if err != nil {
			return err
		}
	}

	// If the test requested some packages to be overridden, we do two things.
	// First, if the package is listed as a direct dependency of the project, we
	// change the version constraint in the package.json. For transitive
	// dependencies, we use npms's "overrides" feature to force them to a
	// specific version.
	if len(pt.Opts.Overrides) > 0 {
		packageJSON, err := ReadPackageJSON(cwd)
		if err != nil {
			return err
		}

		overrides := make(map[string]interface{})

		for packageName, packageVersion := range pt.Opts.Overrides {
			for _, section := range []string{"dependencies", "devDependencies"} {
				if _, has := packageJSON[section]; has {
					entry := packageJSON[section].(map[string]interface{})

					if _, has := entry[packageName]; has {
						entry[packageName] = packageVersion
					}
				}
			}

			pt.T.Logf("adding resolution for %s to version %s", packageName, packageVersion)
			overrides[packageName] = packageVersion
		}

		// Wack any existing resolutions section with our newly computed one.
		packageJSON["overrides"] = overrides

		if err := WritePackageJSON(cwd, packageJSON); err != nil {
			return err
		}
	}

	// Now ensure dependencies are present.
	if err = pt.RunNpmCommand("npm-install", []string{"install"}, cwd); err != nil {
		return err
	}

	if !pt.Opts.RunUpdateTest {
		if err = pt.NpmLinkPackageDeps(cwd); err != nil {
			return err
		}
	}

	if pt.Opts.RunBuild {
		// And finally compile it using whatever build steps are in the package.json file.
		if err = pt.RunNpmCommand("npm-build", []string{"run", "build"}, cwd); err != nil {
			return err
		}
	}

	return nil
}

// ReadPackageJSON unmarshals the package.json file located in pathToPackage.
func ReadPackageJSON(pathToPackage string) (map[string]interface{}, error) {
	f, err := os.Open(filepath.Join(pathToPackage, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("opening package.json: %w", err)
	}
	defer contract.IgnoreClose(f)

	var ret map[string]interface{}
	if err := json.NewDecoder(f).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decoding package.json: %w", err)
	}

	return ret, nil
}

func WritePackageJSON(pathToPackage string, metadata map[string]interface{}) error {
	// os.Create truncates the already existing file.
	f, err := os.Create(filepath.Join(pathToPackage, "package.json"))
	if err != nil {
		return fmt.Errorf("opening package.json: %w", err)
	}
	defer contract.IgnoreClose(f)

	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("writing package.json: %w", err)
	}
	return nil
}

// PreparePythonProject runs setup necessary to get a Python project ready for `pulumi` commands.
func (pt *ProgramTester) PreparePythonProject(projinfo *engine.Projinfo) error {
	cwd, _, err := projinfo.GetPwdMain()
	if err != nil {
		return err
	}

	if pt.Opts.UsePipenv {
		if err = pt.PreparePythonProjectWithPipenv(cwd); err != nil {
			return err
		}
	} else {
		venvPath := "venv"
		if cwd != projinfo.Root {
			venvPath = filepath.Join(cwd, "venv")
		}

		if pt.Opts.GetUseSharedVirtualEnv() {
			requirementsPath := filepath.Join(cwd, "requirements.txt")
			requirementsmd5, err := HashFile(requirementsPath)
			if err != nil {
				return err
			}
			pt.Opts.VirtualEnvDir = fmt.Sprintf("pulumi-venv-%x", requirementsmd5)
			venvPath = filepath.Join(pt.Opts.SharedVirtualEnvPath, pt.Opts.VirtualEnvDir)
		}
		if err = pt.RunPythonCommand("python-venv", []string{"-m", "venv", venvPath}, cwd); err != nil {
			return err
		}

		projinfo.Proj.Runtime.SetOption("virtualenv", venvPath)
		projfile := filepath.Join(projinfo.Root, workspace.ProjectFile+".yaml")
		if err = projinfo.Proj.Save(projfile); err != nil {
			return fmt.Errorf("saving project: %w", err)
		}

		if pt.Opts.InstallDevReleases {
			command := []string{"python", "-m", "pip", "install", "--pre", "pulumi"}
			if err := pt.RunVirtualEnvCommand("virtualenv-pip-install", command, cwd); err != nil {
				return err
			}
		}
		command := []string{"python", "-m", "pip", "install", "-r", "requirements.txt"}
		if err := pt.RunVirtualEnvCommand("virtualenv-pip-install", command, cwd); err != nil {
			return err
		}
	}

	if !pt.Opts.RunUpdateTest {
		if err = pt.InstallPipPackageDeps(cwd); err != nil {
			return err
		}
	}

	return nil
}

func (pt *ProgramTester) PreparePythonProjectWithPipenv(cwd string) error {
	// Allow ENV var based overload of desired Python version for
	// the Pipenv environment. This is useful in CI scenarios that
	// need to pin a specific version such as 3.9.x vs 3.10.x.
	pythonVersion := os.Getenv("PYTHON_VERSION")
	if pythonVersion == "" {
		pythonVersion = "3"
	}

	// Create a new Pipenv environment. This bootstraps a new virtual environment containing the version of Python that
	// we requested. Note that this version of Python is sourced from the machine, so you must first install the version
	// of Python that you are requesting on the host machine before building a virtualenv for it.

	if err := pt.RunPipenvCommand("pipenv-new", []string{"--python", pythonVersion}, cwd); err != nil {
		return err
	}

	// Install the package's dependencies. We do this by running `pip` inside the virtualenv that `pipenv` has created.
	// We don't use `pipenv install` because we don't want a lock file and prefer the similar model of `pip install`
	// which matches what our customers do
	command := []string{"run", "pip", "install", "-r", "requirements.txt"}
	if pt.Opts.InstallDevReleases {
		command = []string{"run", "pip", "install", "--pre", "-r", "requirements.txt"}
	}
	err := pt.RunPipenvCommand("pipenv-install", command, cwd)
	if err != nil {
		return err
	}
	return nil
}

// NpmLinkPackageDeps bring in package dependencies via npm
func (pt *ProgramTester) NpmLinkPackageDeps(cwd string) error {
	for _, dependency := range pt.Opts.Dependencies {
		if err := pt.RunNpmCommand("npm-link", []string{"link", dependency}, cwd); err != nil {
			return err
		}
	}

	return nil
}

// InstallPipPackageDeps brings in package dependencies via pip install
func (pt *ProgramTester) InstallPipPackageDeps(cwd string) error {
	var err error
	for _, dep := range pt.Opts.Dependencies {
		// If the given filepath isn't absolute, make it absolute. We're about to pass it to pipenv and pipenv is
		// operating inside of a random folder in /tmp.
		if !filepath.IsAbs(dep) {
			dep, err = filepath.Abs(dep)
			if err != nil {
				return err
			}
		}

		if pt.Opts.UsePipenv {
			if err := pt.RunPipenvCommand("pipenv-install-package",
				[]string{"run", "pip", "install", "-e", dep}, cwd); err != nil {
				return err
			}
		} else {
			if err := pt.RunVirtualEnvCommand("virtualenv-pip-install-package",
				[]string{"python", "-m", "pip", "install", "-e", dep}, cwd); err != nil {
				return err
			}
		}
	}

	return nil
}

func GetVirtualenvBinPath(cwd, bin string, pt *ProgramTester) (string, error) {
	virtualEnvBasePath := filepath.Join(cwd, pt.Opts.VirtualEnvDir)
	if pt.Opts.GetUseSharedVirtualEnv() {
		virtualEnvBasePath = filepath.Join(pt.Opts.SharedVirtualEnvPath, pt.Opts.VirtualEnvDir)
	}
	virtualenvBinPath := filepath.Join(virtualEnvBasePath, "bin", bin)
	if runtime.GOOS == WindowsOS {
		virtualenvBinPath = filepath.Join(virtualEnvBasePath, "Scripts", bin+".exe")
	}
	if info, err := os.Stat(virtualenvBinPath); err != nil || info.IsDir() {
		return "", fmt.Errorf("Expected %s to exist in virtual environment at %q", bin, virtualenvBinPath)
	}
	return virtualenvBinPath, nil
}

// getSanitizedPkg strips the version string from a go dep
// Note: most of the pulumi modules don't use major version subdirectories for modules
func GetSanitizedModulePath(pkg string) string {
	re := regexp.MustCompile(`v\d`)
	v := re.FindString(pkg)
	if v != "" {
		return strings.TrimSuffix(strings.ReplaceAll(pkg, v, ""), "/")
	}
	return pkg
}

func GetRewritePath(pkg string, gopath string, depRoot string) string {
	var depParts []string
	sanitizedPkg := GetSanitizedModulePath(pkg)

	splitPkg := strings.Split(sanitizedPkg, "/")

	if depRoot != "" {
		// Get the package name
		// This is the value after "github.com/foo/bar"
		repoName := splitPkg[2]
		basePath := splitPkg[len(splitPkg)-1]
		if basePath == repoName {
			depParts = []string{depRoot, repoName}
		} else {
			depParts = []string{depRoot, repoName, basePath}
		}
		return filepath.Join(depParts...)
	}
	depParts = append([]string{gopath, "src"}, splitPkg...)
	return filepath.Join(depParts...)
}

// Fetchs the GOPATH
func GoPath() (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		usr, userErr := user.Current()
		if userErr != nil {
			return "", userErr
		}
		gopath = filepath.Join(usr.HomeDir, "go")
	}
	return gopath, nil
}

// PrepareGoProject runs setup necessary to get a Go project ready for `pulumi` commands.
func (pt *ProgramTester) PrepareGoProject(projinfo *engine.Projinfo) error {
	// Go programs are compiled, so we will compile the project first.
	goBin, err := pt.GetGoBin()
	if err != nil {
		return fmt.Errorf("locating `go` binary: %w", err)
	}

	depRoot := os.Getenv("PULUMI_GO_DEP_ROOT")
	gopath, userError := GoPath()
	if userError != nil {
		return fmt.Errorf("error getting GOPATH: %w", userError)
	}

	cwd, _, err := projinfo.GetPwdMain()
	if err != nil {
		return fmt.Errorf("error getting project working directory: %w", err)
	}

	// initialize a go.mod for dependency resolution if one doesn't exist
	_, err = os.Stat(filepath.Join(cwd, "go.mod"))
	if err != nil {
		err = pt.RunCommand("go-mod-init", []string{goBin, "mod", "init"}, cwd)
		if err != nil {
			return fmt.Errorf("error initializing go.mod: %w", err)
		}
	}

	// install dev dependencies if requested
	if pt.Opts.InstallDevReleases {
		// We're currently only installing pulumi/pulumi dependencies, which always have
		// "master" as the default branch.
		defaultBranch := "master"
		err = pt.RunCommand("go-get-dev-deps", []string{
			goBin, "get", "-u", "github.com/pulumi/pulumi/sdk/v3@" + defaultBranch,
		}, cwd)
		if err != nil {
			return fmt.Errorf("error installing dev dependencies: %w", err)
		}
	}

	// link local dependencies
	for _, dep := range pt.Opts.Dependencies {
		editStr, err := GetEditStr(dep, gopath, depRoot)
		if err != nil {
			return fmt.Errorf("error generating go mod replacement for dep %q: %w", dep, err)
		}
		err = pt.RunCommand("go-mod-edit", []string{goBin, "mod", "edit", "-replace", editStr}, cwd)
		if err != nil {
			return fmt.Errorf("error adding go mod replacement for dep %q: %w", dep, err)
		}
	}

	// tidy to resolve all transitive dependencies including from local dependencies above.
	err = pt.RunCommand("go-mod-tidy", []string{goBin, "mod", "tidy"}, cwd)
	if err != nil {
		return fmt.Errorf("error running go mod tidy: %w", err)
	}

	if pt.Opts.RunBuild {
		outBin := filepath.Join(gopath, "bin", string(projinfo.Proj.Name))
		if runtime.GOOS == WindowsOS {
			outBin = outBin + ".exe"
		}
		err = pt.RunCommand("go-build", []string{goBin, "build", "-o", outBin, "."}, cwd)
		if err != nil {
			return fmt.Errorf("error building application: %w", err)
		}

		_, err = os.Stat(outBin)
		if err != nil {
			return fmt.Errorf("error finding built application artifact: %w", err)
		}
	}

	return nil
}

func GetEditStr(dep string, gopath string, depRoot string) (string, error) {
	checkModName := true
	var err error
	var replacedModName string
	var targetModDir string
	if strings.ContainsRune(dep, '=') {
		parts := strings.Split(dep, "=")
		replacedModName = parts[0]
		targetModDir = parts[1]
	} else if !modfile.IsDirectoryPath(dep) {
		replacedModName = dep
		targetModDir = GetRewritePath(dep, gopath, depRoot)
	} else {
		targetModDir = dep
		replacedModName, err = GetModName(targetModDir)
		if err != nil {
			return "", err
		}
		// We've read the package name from the go.mod file, skip redundant check below.
		checkModName = false
	}

	targetModDir, err = filepath.Abs(targetModDir)
	if err != nil {
		return "", err
	}

	if checkModName {
		targetModName, err := GetModName(targetModDir)
		if err != nil {
			return "", fmt.Errorf("no go.mod at directory, set the path to the module explicitly or place "+
				"the dependency in the path specified by PULUMI_GO_DEP_ROOT or the default GOPATH: %w", err)
		}
		targetPrefix, _, ok := module.SplitPathVersion(targetModName)
		if !ok {
			return "", fmt.Errorf("invalid module path for target module %q", targetModName)
		}
		replacedPrefix, _, ok := module.SplitPathVersion(replacedModName)
		if !ok {
			return "", fmt.Errorf("invalid module path for replaced module %q", replacedModName)
		}
		if targetPrefix != replacedPrefix {
			return "", fmt.Errorf("found module path with prefix %s, expected %s", targetPrefix, replacedPrefix)
		}
	}

	editStr := fmt.Sprintf("%s=%s", replacedModName, targetModDir)
	return editStr, nil
}

func GetModName(dir string) (string, error) {
	pkgModPath := filepath.Join(dir, "go.mod")
	pkgModData, err := os.ReadFile(pkgModPath)
	if err != nil {
		return "", fmt.Errorf("error reading go.mod at %s: %w", dir, err)
	}
	pkgMod, err := modfile.Parse(pkgModPath, pkgModData, nil)
	if err != nil {
		return "", fmt.Errorf("error parsing go.mod at %s: %w", dir, err)
	}

	return pkgMod.Module.Mod.Path, nil
}

// PrepareDotNetProject runs setup necessary to get a .NET project ready for `pulumi` commands.
func (pt *ProgramTester) PrepareDotNetProject(projinfo *engine.Projinfo) error {
	dotNetBin, err := pt.GetDotNetBin()
	if err != nil {
		return fmt.Errorf("locating `dotnet` binary: %w", err)
	}

	cwd, _, err := projinfo.GetPwdMain()
	if err != nil {
		return err
	}

	localNuget := os.Getenv("PULUMI_LOCAL_NUGET")
	if localNuget == "" {
		home := os.Getenv("HOME")
		localNuget = filepath.Join(home, ".pulumi-dev", "nuget")
	}

	if pt.Opts.InstallDevReleases {
		err = pt.RunCommand("dotnet-add-package",
			[]string{
				dotNetBin, "add", "package", "Pulumi",
				"--prerelease",
			},
			cwd)
		if err != nil {
			return err
		}
	}

	for _, dep := range pt.Opts.Dependencies {
		// dotnet add package requires a specific version in case of a pre-release, so we have to look it up.
		globPattern := filepath.Join(localNuget, dep+".?.*.nupkg")
		matches, err := filepath.Glob(globPattern)
		if err != nil {
			return fmt.Errorf("failed to find a local Pulumi NuGet package: %w", err)
		}
		if len(matches) != 1 {
			return fmt.Errorf("attempting to find a local NuGet package %s by searching %s yielded %d results: %v",
				dep,
				globPattern,
				len(matches),
				matches)
		}
		file := filepath.Base(matches[0])
		r := strings.NewReplacer(dep+".", "", ".nupkg", "")
		version := r.Replace(file)

		// We don't restore because the program might depend on external
		// packages which cannot be found in our local nuget source. A restore
		// will happen automatically as part of the `pulumi up`.
		err = pt.RunCommand("dotnet-add-package",
			[]string{
				dotNetBin, "add", "package", dep,
				"-v", version,
				"-s", localNuget,
				"--no-restore",
			},
			cwd)
		if err != nil {
			return fmt.Errorf("failed to add dependency on %s: %w", dep, err)
		}
	}

	return nil
}

func (pt *ProgramTester) PrepareYAMLProject(projinfo *engine.Projinfo) error {
	// YAML doesn't need any system setup, and should auto-install required plugins
	return nil
}

func (pt *ProgramTester) PrepareJavaProject(projinfo *engine.Projinfo) error {
	// Java doesn't need any system setup, and should auto-install required plugins
	return nil
}

func (pt *ProgramTester) DefaultPrepareProject(projinfo *engine.Projinfo) error {
	// Based on the language, invoke the right routine to prepare the target directory.
	switch rt := projinfo.Proj.Runtime.Name(); rt {
	case NodeJSRuntime:
		return pt.PrepareNodeJSProject(projinfo)
	case PythonRuntime:
		return pt.PreparePythonProject(projinfo)
	case GoRuntime:
		return pt.PrepareGoProject(projinfo)
	case DotNetRuntime:
		return pt.PrepareDotNetProject(projinfo)
	case YAMLRuntime:
		return pt.PrepareYAMLProject(projinfo)
	case JavaRuntime:
		return pt.PrepareJavaProject(projinfo)
	default:
		return fmt.Errorf("unrecognized project runtime: %s", rt)
	}
}

// AssertPerfBenchmark implements the integration.TestStatsReporter interface, and reports test
// failures when a scenario exceeds the provided threshold.
type AssertPerfBenchmark struct {
	T                  *testing.T
	MaxPreviewDuration time.Duration
	MaxUpdateDuration  time.Duration
}

func (t AssertPerfBenchmark) ReportCommand(stats TestCommandStats) {
	var maxDuration *time.Duration
	if strings.HasPrefix(stats.StepName, "pulumi-preview") {
		maxDuration = &t.MaxPreviewDuration
	}
	if strings.HasPrefix(stats.StepName, "pulumi-update") {
		maxDuration = &t.MaxUpdateDuration
	}

	if maxDuration != nil && *maxDuration != 0 {
		if stats.ElapsedSeconds < maxDuration.Seconds() {
			t.T.Logf(
				"Test step %q was under threshold. %.2fs (max %.2fs)",
				stats.StepName, stats.ElapsedSeconds, maxDuration.Seconds())
		} else {
			t.T.Errorf(
				"Test step %q took longer than expected. %.2fs vs. max %.2fs",
				stats.StepName, stats.ElapsedSeconds, maxDuration.Seconds())
		}
	}
}
