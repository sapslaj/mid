package pdiff_test

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/pkg/pdiff"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/provider/midtypes"
	"github.com/sapslaj/mid/provider/resource"
)

func TestDiffAttributes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputs     any
		state      any
		attributes []string
		expect     p.DiffResponse
	}{
		"anon struct unchanged": {
			inputs: struct {
				Ignore    string `pulumi:"ignore"`
				Unchanged string `pulumi:"unchanged"`
			}{
				Ignore:    "inputs",
				Unchanged: "unchanged",
			},
			state: struct {
				Ignore    string `pulumi:"ignore"`
				Unchanged string `pulumi:"unchanged"`
				Extra     string `pulumi:"extra"`
			}{
				Ignore:    "state",
				Unchanged: "unchanged",
				Extra:     "extra",
			},
			expect: p.DiffResponse{
				HasChanges:          false,
				DeleteBeforeReplace: false,
				DetailedDiff:        map[string]p.PropertyDiff{},
			},
		},

		"anon struct changed": {
			inputs: struct {
				Ignore    string `pulumi:"ignore"`
				Changed   string `pulumi:"changed"`
				Unchanged string `pulumi:"unchanged"`
			}{
				Ignore:    "inputs",
				Changed:   "changed",
				Unchanged: "unchanged",
			},
			state: struct {
				Ignore    string `pulumi:"ignore"`
				Changed   string `pulumi:"changed"`
				Unchanged string `pulumi:"unchanged"`
				Extra     string `pulumi:"extra"`
			}{
				Ignore:    "state",
				Changed:   "unchanged",
				Unchanged: "unchanged",
				Extra:     "extra",
			},
			attributes: []string{
				"changed",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"changed": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"File no diff": {
			inputs: resource.FileArgs{
				Path:    "/testing",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("unchanged"),
				Triggers: &midtypes.TriggersInput{
					Refresh: &[]any{1},
				},
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
					Triggers: &midtypes.TriggersInput{
						Refresh: &[]any{1},
					},
				},
				BackupFile: ptr.Of("/testing.backup"),
				Triggers: midtypes.TriggersOutput{
					LastChanged: "now",
					Refresh:     &[]any{1},
				},
			},
			attributes: []string{
				"path",
				"ensure",
				"owner",
				"content",
			},
			expect: p.DiffResponse{
				HasChanges:          false,
				DeleteBeforeReplace: false,
				DetailedDiff:        map[string]p.PropertyDiff{},
			},
		},

		"File non-replacement diff": {
			inputs: resource.FileArgs{
				Path:    "/testing",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("changed"),
				Triggers: &midtypes.TriggersInput{
					Refresh: &[]any{1},
				},
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
					Triggers: &midtypes.TriggersInput{
						Refresh: &[]any{1},
					},
				},
				BackupFile: ptr.Of("/testing.backup"),
				Triggers: midtypes.TriggersOutput{
					LastChanged: "now",
					Refresh:     &[]any{1},
				},
			},
			attributes: []string{
				"path",
				"ensure",
				"owner",
				"content",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"content": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"File replacement diff": {
			inputs: resource.FileArgs{
				Path:    "/changed",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("changed"),
				Triggers: &midtypes.TriggersInput{
					Refresh: &[]any{1},
				},
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
					Triggers: &midtypes.TriggersInput{
						Refresh: &[]any{1},
					},
				},
				BackupFile: ptr.Of("/testing.backup"),
				Triggers: midtypes.TriggersOutput{
					LastChanged: "now",
					Refresh:     &[]any{1},
				},
			},
			attributes: []string{
				"path",
				"ensure",
				"owner",
				"content",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"path": {
						Kind:      p.UpdateReplace,
						InputDiff: true,
					},
					"content": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"deeply nested attributes": {
			inputs: struct {
				First struct {
					Second []struct {
						Third string `pulumi:"third"`
					} `pulumi:"second"`
				} `pulumi:"first"`
			}{
				First: struct {
					Second []struct {
						Third string `pulumi:"third"`
					} `pulumi:"second"`
				}{
					Second: []struct {
						Third string `pulumi:"third"`
					}{
						{
							Third: "changed",
						},
					},
				},
			},
			state: struct {
				First struct {
					Second []struct {
						Third string `pulumi:"third"`
					} `pulumi:"second"`
				} `pulumi:"first"`
			}{
				First: struct {
					Second []struct {
						Third string `pulumi:"third"`
					} `pulumi:"second"`
				}{
					Second: []struct {
						Third string `pulumi:"third"`
					}{
						{
							Third: "unchanged",
						},
					},
				},
			},
			attributes: []string{
				"first",
			},
			expect: p.DiffResponse{
				HasChanges: true,
				DetailedDiff: map[string]p.PropertyDiff{
					"first.second[0].third": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pdiff.DiffAttributes(tc.inputs, tc.state, tc.attributes)

			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestDiffAllAttributesExcept(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputs           any
		state            any
		exceptAttributes []string
		expect           p.DiffResponse
	}{
		"anon struct unchanged": {
			inputs: struct {
				Ignore    string `pulumi:"ignore"`
				Unchanged string `pulumi:"unchanged"`
			}{
				Ignore:    "inputs",
				Unchanged: "unchanged",
			},
			state: struct {
				Ignore    string `pulumi:"ignore"`
				Unchanged string `pulumi:"unchanged"`
				Extra     string `pulumi:"extra"`
			}{
				Ignore:    "state",
				Unchanged: "unchanged",
				Extra:     "extra",
			},
			exceptAttributes: []string{
				"ignore",
			},
			expect: p.DiffResponse{
				HasChanges:          false,
				DeleteBeforeReplace: false,
				DetailedDiff:        map[string]p.PropertyDiff{},
			},
		},

		"anon struct changed": {
			inputs: struct {
				Ignore    string `pulumi:"ignore"`
				Changed   string `pulumi:"changed"`
				Unchanged string `pulumi:"unchanged"`
			}{
				Ignore:    "inputs",
				Changed:   "changed",
				Unchanged: "unchanged",
			},
			state: struct {
				Ignore    string `pulumi:"ignore"`
				Changed   string `pulumi:"changed"`
				Unchanged string `pulumi:"unchanged"`
				Extra     string `pulumi:"extra"`
			}{
				Ignore:    "state",
				Changed:   "unchanged",
				Unchanged: "unchanged",
				Extra:     "extra",
			},
			exceptAttributes: []string{
				"ignore",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"changed": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"File no diff": {
			inputs: resource.FileArgs{
				Path:    "/testing",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("unchanged"),
				Triggers: &midtypes.TriggersInput{
					Refresh: &[]any{1},
				},
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
					Triggers: &midtypes.TriggersInput{
						Refresh: &[]any{1},
					},
				},
				BackupFile: ptr.Of("/testing.backup"),
				Triggers: midtypes.TriggersOutput{
					LastChanged: "now",
					Refresh:     &[]any{1},
				},
			},
			exceptAttributes: []string{
				"triggers",
			},
			expect: p.DiffResponse{
				HasChanges:          false,
				DeleteBeforeReplace: false,
				DetailedDiff:        map[string]p.PropertyDiff{},
			},
		},

		"File non-replacement diff": {
			inputs: resource.FileArgs{
				Path:    "/testing",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("changed"),
				Triggers: &midtypes.TriggersInput{
					Refresh: &[]any{1},
				},
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
					Triggers: &midtypes.TriggersInput{
						Refresh: &[]any{1},
					},
				},
				BackupFile: ptr.Of("/testing.backup"),
				Triggers: midtypes.TriggersOutput{
					LastChanged: "now",
					Refresh:     &[]any{1},
				},
			},
			exceptAttributes: []string{
				"triggers",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"content": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"File replacement diff": {
			inputs: resource.FileArgs{
				Path:    "/changed",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("changed"),
				Triggers: &midtypes.TriggersInput{
					Refresh: &[]any{1},
				},
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
					Triggers: &midtypes.TriggersInput{
						Refresh: &[]any{1},
					},
				},
				BackupFile: ptr.Of("/testing.backup"),
				Triggers: midtypes.TriggersOutput{
					LastChanged: "now",
					Refresh:     &[]any{1},
				},
			},
			exceptAttributes: []string{
				"triggers",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"path": {
						Kind:      p.UpdateReplace,
						InputDiff: true,
					},
					"content": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pdiff.DiffAllAttributesExcept(tc.inputs, tc.state, tc.exceptAttributes)

			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestDiffAllAttributes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputs any
		state  any
		expect p.DiffResponse
	}{
		"anon struct unchanged": {
			inputs: struct {
				Unchanged string `pulumi:"unchanged"`
			}{
				Unchanged: "unchanged",
			},
			state: struct {
				Unchanged string `pulumi:"unchanged"`
				Extra     string `pulumi:"extra"`
			}{
				Unchanged: "unchanged",
				Extra:     "extra",
			},
			expect: p.DiffResponse{
				HasChanges:          false,
				DeleteBeforeReplace: false,
				DetailedDiff:        map[string]p.PropertyDiff{},
			},
		},

		"anon struct changed": {
			inputs: struct {
				Changed   string `pulumi:"changed"`
				Unchanged string `pulumi:"unchanged"`
			}{
				Changed:   "changed",
				Unchanged: "unchanged",
			},
			state: struct {
				Changed   string `pulumi:"changed"`
				Unchanged string `pulumi:"unchanged"`
				Extra     string `pulumi:"extra"`
			}{
				Changed:   "unchanged",
				Unchanged: "unchanged",
				Extra:     "extra",
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"changed": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"File no diff": {
			inputs: resource.FileArgs{
				Path:    "/testing",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("unchanged"),
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
				},
				BackupFile: ptr.Of("/testing.backup"),
			},
			expect: p.DiffResponse{
				HasChanges:          false,
				DeleteBeforeReplace: false,
				DetailedDiff:        map[string]p.PropertyDiff{},
			},
		},

		"File non-replacement diff": {
			inputs: resource.FileArgs{
				Path:    "/testing",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("changed"),
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
				},
				BackupFile: ptr.Of("/testing.backup"),
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"content": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"File replacement diff": {
			inputs: resource.FileArgs{
				Path:    "/changed",
				Ensure:  ptr.Of(resource.FileEnsureFile),
				Owner:   ptr.Of("nobody"),
				Content: ptr.Of("changed"),
			},
			state: resource.FileState{
				FileArgs: resource.FileArgs{
					Path:    "/testing",
					Ensure:  ptr.Of(resource.FileEnsureFile),
					Owner:   ptr.Of("nobody"),
					Content: ptr.Of("unchanged"),
				},
				BackupFile: ptr.Of("/testing.backup"),
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"path": {
						Kind:      p.UpdateReplace,
						InputDiff: true,
					},
					"content": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"AnsibleTaskList": {
			inputs: resource.AnsibleTaskListArgs{
				Tasks: resource.AnsibleTaskListArgsTasks{
					Create: []resource.AnsibleTaskListArgsTask{
						{
							Module: "file",
							Args: map[string]any{
								"path":  "/testing",
								"state": "touch",
							},
						},
						{
							Module: "blockinfile",
							Args: map[string]any{
								"path":  "/testing",
								"state": "present",
								"block": "updating",
							},
						},
					},
					Update: &[]resource.AnsibleTaskListArgsTask{
						{
							Module: "blockinfile",
							Args: map[string]any{
								"path":  "/testing",
								"state": "present",
								"block": "updating",
							},
						},
					},
					Delete: &[]resource.AnsibleTaskListArgsTask{
						{
							Module: "file",
							Args: map[string]any{
								"path":  "/testing",
								"state": "absent",
							},
						},
					},
				},
			},
			state: resource.AnsibleTaskListState{
				AnsibleTaskListArgs: resource.AnsibleTaskListArgs{
					Tasks: resource.AnsibleTaskListArgsTasks{
						Create: []resource.AnsibleTaskListArgsTask{
							{
								Module: "file",
								Args: map[string]any{
									"path":  "/testing",
									"state": "touch",
								},
							},
							{
								Module: "blockinfile",
								Args: map[string]any{
									"path":  "/testing",
									"state": "present",
									"block": "creating",
								},
							},
						},
						Update: &[]resource.AnsibleTaskListArgsTask{
							{
								Module: "blockinfile",
								Args: map[string]any{
									"path":  "/testing",
									"state": "present",
									"block": "creating",
								},
							},
						},
						Delete: &[]resource.AnsibleTaskListArgsTask{
							{
								Module: "file",
								Args: map[string]any{
									"path":  "/testing",
									"state": "absent",
								},
							},
						},
					},
				},
				Results: resource.AnsibleTaskListStateResults{
					Lifecycle: "create",
					Tasks: []resource.AnsibleTaskListStateTaskResult{
						{
							AnsibleTaskListArgsTask: resource.AnsibleTaskListArgsTask{
								Module: "file",
								Args: map[string]any{
									"path":  "/testing",
									"state": "touch",
								},
							},
							Success: true,
						},
						{
							AnsibleTaskListArgsTask: resource.AnsibleTaskListArgsTask{
								Module: "blockinfile",
								Args: map[string]any{
									"path":  "/testing",
									"state": "present",
									"block": "creating",
								},
							},
							Success: true,
						},
					},
				},
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: false,
				DetailedDiff: map[string]p.PropertyDiff{
					"tasks.create[1].args.block": {
						Kind:      p.Update,
						InputDiff: true,
					},
					"tasks.update[0].args.block": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pdiff.DiffAllAttributes(tc.inputs, tc.state)

			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestForceDiffReplace(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		diff   p.DiffResponse
		expect p.DiffResponse
	}{
		"gets expected result": {
			diff: p.DiffResponse{
				HasChanges: true,
				DetailedDiff: map[string]p.PropertyDiff{
					"add": {
						Kind: p.Add,
					},
					"addreplace": {
						Kind: p.AddReplace,
					},
					"delete": {
						Kind: p.Delete,
					},
					"deletereplace": {
						Kind: p.DeleteReplace,
					},
					"update": {
						Kind: p.Update,
					},
					"updatereplace": {
						Kind: p.UpdateReplace,
					},
					"stable": {
						Kind: p.Stable,
					},
				},
			},
			expect: p.DiffResponse{
				HasChanges: true,
				DetailedDiff: map[string]p.PropertyDiff{
					"add": {
						Kind: p.AddReplace,
					},
					"addreplace": {
						Kind: p.AddReplace,
					},
					"delete": {
						Kind: p.DeleteReplace,
					},
					"deletereplace": {
						Kind: p.DeleteReplace,
					},
					"update": {
						Kind: p.UpdateReplace,
					},
					"updatereplace": {
						Kind: p.UpdateReplace,
					},
					"stable": {
						Kind: p.Stable,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pdiff.ForceDiffReplace(tc.diff)

			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestMergeDiffResponses(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		diffs  []p.DiffResponse
		expect p.DiffResponse
	}{
		"deep merge": {
			diffs: []p.DiffResponse{
				{
					HasChanges: true,
					DetailedDiff: map[string]p.PropertyDiff{
						"foo": {
							Kind:      p.Add,
							InputDiff: true,
						},
					},
				},
				{
					HasChanges: true,
					DetailedDiff: map[string]p.PropertyDiff{
						"bar": {
							Kind:      p.Add,
							InputDiff: true,
						},
						"baz": {
							Kind:      p.Add,
							InputDiff: true,
						},
					},
				},
				{
					HasChanges: true,
					DetailedDiff: map[string]p.PropertyDiff{
						"baz": {
							Kind:      p.Update,
							InputDiff: true,
						},
					},
				},
			},
			expect: p.DiffResponse{
				HasChanges: true,
				DetailedDiff: map[string]p.PropertyDiff{
					"foo": {
						Kind:      p.Add,
						InputDiff: true,
					},
					"bar": {
						Kind:      p.Add,
						InputDiff: true,
					},
					"baz": {
						Kind:      p.Update,
						InputDiff: true,
					},
				},
			},
		},

		"has changes": {
			diffs: []p.DiffResponse{
				{
					HasChanges: false,
				},
				{
					HasChanges: true,
					DetailedDiff: map[string]p.PropertyDiff{
						"foo": {
							Kind: p.Add,
						},
					},
				},
				{
					HasChanges:   false,
					DetailedDiff: map[string]p.PropertyDiff{},
				},
			},
			expect: p.DiffResponse{
				HasChanges: true,
				DetailedDiff: map[string]p.PropertyDiff{
					"foo": {
						Kind: p.Add,
					},
				},
			},
		},

		"delete before replace": {
			diffs: []p.DiffResponse{
				{
					HasChanges:          false,
					DeleteBeforeReplace: false,
				},
				{
					HasChanges:          true,
					DeleteBeforeReplace: false,
					DetailedDiff: map[string]p.PropertyDiff{
						"foo": {
							Kind: p.Add,
						},
					},
				},
				{
					HasChanges:          false,
					DeleteBeforeReplace: true,
					DetailedDiff:        map[string]p.PropertyDiff{},
				},
				{
					HasChanges:          false,
					DeleteBeforeReplace: false,
					DetailedDiff:        map[string]p.PropertyDiff{},
				},
			},
			expect: p.DiffResponse{
				HasChanges:          true,
				DeleteBeforeReplace: true,
				DetailedDiff: map[string]p.PropertyDiff{
					"foo": {
						Kind: p.Add,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := pdiff.MergeDiffResponses(tc.diffs...)

			assert.Equal(t, tc.expect, got)
		})
	}
}
