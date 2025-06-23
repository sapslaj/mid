package resource

import (
	"testing"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/stretchr/testify/assert"
)

func TestFile_updateStateDrifted(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputs        FileArgs
		props         []string
		expectDrifted []string
	}{
		"no intersection": {
			inputs: FileArgs{
				Path: "/foo",
			},
			props: []string{
				"accessTime",
			},
			expectDrifted: []string{},
		},

		"props is subset of inputs": {
			inputs: FileArgs{
				Path:   "/foo",
				Backup: ptr.Of(true),
			},
			props: []string{
				"backup",
			},
			expectDrifted: []string{
				"backup",
			},
		},

		"inputs is a subset of props": {
			inputs: FileArgs{
				Path: "/foo",
			},
			props: []string{
				"accessTime",
				"backup",
				"path",
			},
			expectDrifted: []string{
				"path",
			},
		},

		"inputs intersects with props": {
			inputs: FileArgs{
				Path:   "/foo",
				Ensure: ptr.Of(FileEnsureHard),
			},
			props: []string{
				"accessTime",
				"ensure",
			},
			expectDrifted: []string{
				"ensure",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := File{}

			state := r.updateStateDrifted(tc.inputs, FileState{FileArgs: tc.inputs}, tc.props)

			assert.ElementsMatch(t, tc.expectDrifted, state.Drifted)
		})
	}
}

func TestFile_ansibleFileDiffedAttributes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fileReturn ansible.FileReturn
		expect     []string
	}{
		"no diff": {
			fileReturn: ansible.FileReturn{
				AnsibleCommonReturns: ansible.AnsibleCommonReturns{
					Changed: false,
					Diff: ptr.ToAny(ptr.Of(map[string]any{
						"before": map[string]any{
							"path": "/foo",
						},
						"after": map[string]any{
							"path": "/foo",
						},
					})),
				},
			},
			expect: []string{},
		},

		"diff": {
			fileReturn: ansible.FileReturn{
				AnsibleCommonReturns: ansible.AnsibleCommonReturns{
					Changed: false,
					Diff: ptr.ToAny(ptr.Of(map[string]any{
						"before": map[string]any{
							"path":  "/foo",
							"owner": "root",
						},
						"after": map[string]any{
							"path":  "/foo",
							"owner": "games",
						},
					})),
				},
			},
			expect: []string{
				"owner",
			},
		},

		"diff value and type": {
			fileReturn: ansible.FileReturn{
				AnsibleCommonReturns: ansible.AnsibleCommonReturns{
					Changed: false,
					Diff: ptr.ToAny(ptr.Of(map[string]any{
						"before": map[string]any{
							"path": "/foo",
							"mode": 777,
						},
						"after": map[string]any{
							"path": "/foo",
							"mode": "a=rwx",
						},
					})),
				},
			},
			expect: []string{
				"mode",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := File{}

			got := r.ansibleFileDiffedAttributes(tc.fileReturn)

			assert.ElementsMatch(t, tc.expect, got)
		})
	}
}
