package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/ptr"
)

func TestApt_argsToTaskParameters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input                   AptArgs
		expected                ansible.AptParameters
		taskParametersNeedsName bool
		canAssumeEnsure         bool
	}{
		"install single package": {
			input: AptArgs{
				Name: ptr.Of("vim"),
			},
			expected: ansible.AptParameters{
				Name:        ptr.Of([]string{"vim"}),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"install single package but in a list": {
			input: AptArgs{
				Names: ptr.Of([]string{"vim"}),
			},
			expected: ansible.AptParameters{
				Name:        ptr.Of([]string{"vim"}),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"install multiple packages": {
			input: AptArgs{
				Names: ptr.Of([]string{"vim", "emacs"}),
			},
			expected: ansible.AptParameters{
				Name:        ptr.Of([]string{"vim", "emacs"}),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"update repositories and install package": {
			input: AptArgs{
				Name:        ptr.Of("vim"),
				UpdateCache: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Name:        ptr.Of([]string{"vim"}),
				UpdateCache: ptr.Of(true),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         true,
		},
		"remove package": {
			input: AptArgs{
				Name:   ptr.Of("emacs"),
				Ensure: ptr.Of("absent"),
			},
			expected: ansible.AptParameters{
				Name:        ptr.Of([]string{"emacs"}),
				State:       ansible.OptionalAptState("absent"),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"allow downgrade": {
			input: AptArgs{
				Name:           ptr.Of("emacs"),
				AllowDowngrade: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Name:           ptr.Of([]string{"emacs"}),
				AllowDowngrade: ptr.Of(true),
				LockTimeout:    ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"fail on autoremove": {
			input: AptArgs{
				Name:             ptr.Of("vim"),
				FailOnAutoremove: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Name:             ptr.Of([]string{"vim"}),
				FailOnAutoremove: ptr.Of(true),
				LockTimeout:      ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"install recommends": {
			input: AptArgs{
				Name:              ptr.Of("vim"),
				InstallRecommends: ptr.Of(false),
			},
			expected: ansible.AptParameters{
				Name:              ptr.Of([]string{"vim"}),
				InstallRecommends: ptr.Of(false),
				LockTimeout:       ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"update all packages": {
			input: AptArgs{
				Name:   ptr.Of("*"),
				Ensure: ptr.Of("latest"),
			},
			expected: ansible.AptParameters{
				Name:        ptr.Of([]string{"*"}),
				State:       ansible.OptionalAptState("latest"),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: true,
			canAssumeEnsure:         true,
		},
		"apt dist-upgrade": {
			input: AptArgs{
				Upgrade: ptr.Of("dist"),
			},
			expected: ansible.AptParameters{
				Upgrade:     ansible.OptionalAptUpgrade("dist"),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"apt update": {
			input: AptArgs{
				UpdateCache: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				UpdateCache: ptr.Of(true),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"apt update with cache valid time": {
			input: AptArgs{
				UpdateCache:    ptr.Of(true),
				CacheValidTime: ptr.Of(3600),
			},
			expected: ansible.AptParameters{
				UpdateCache:    ptr.Of(true),
				CacheValidTime: ptr.Of(3600),
				LockTimeout:    ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"update and upgrade with dpkg options": {
			input: AptArgs{
				Upgrade:     ptr.Of("dist"),
				UpdateCache: ptr.Of(true),
				DpkgOptions: ptr.Of("force-confold,force-confdef"),
			},
			expected: ansible.AptParameters{
				Upgrade:     ansible.OptionalAptUpgrade("dist"),
				UpdateCache: ptr.Of(true),
				DpkgOptions: ptr.Of("force-confold,force-confdef"),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"install from URL": {
			input: AptArgs{
				Deb: ptr.Of("https://ubuntu.pkgs.org/24.04/ubuntu-universe-amd64/neovim_0.9.5-6ubuntu2_amd64.deb.html"),
			},
			expected: ansible.AptParameters{
				Deb:         ptr.Of("https://ubuntu.pkgs.org/24.04/ubuntu-universe-amd64/neovim_0.9.5-6ubuntu2_amd64.deb.html"),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         true,
		},
		"autoclean": {
			input: AptArgs{
				Autoclean: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Autoclean:   ptr.Of(true),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"autoremove": {
			input: AptArgs{
				Autoremove: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Autoremove:  ptr.Of(true),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"autoremove and purge": {
			input: AptArgs{
				Autoremove: ptr.Of(true),
				Purge:      ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Autoremove:  ptr.Of(true),
				Purge:       ptr.Of(true),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
		"clean": {
			input: AptArgs{
				Clean: ptr.Of(true),
			},
			expected: ansible.AptParameters{
				Clean:       ptr.Of(true),
				LockTimeout: ptr.Of(120),
			},
			taskParametersNeedsName: false,
			canAssumeEnsure:         false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := Apt{}

			got, err := r.argsToTaskParameters(tc.input)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)

			taskParametersNeedsName := r.taskParametersNeedsName(tc.input)
			assert.Equal(t, tc.taskParametersNeedsName, taskParametersNeedsName)

			canAssumeEnsure := r.canAssumeEnsure(tc.input)
			assert.Equal(t, tc.canAssumeEnsure, canAssumeEnsure)
		})
	}
}
