// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manages `apt` packages (such as for Debian/Ubuntu).
const AptName = "apt"

// Indicates the desired package state. `latest` ensures that the latest version
// is installed. `build-dep` ensures the package build dependencies are
// installed. `fixed` attempt to correct a system with broken dependencies in
// place.
type AptState string

const (
	AptStateAbsent   AptState = "absent"
	AptStateBuildDep AptState = "build-dep"
	AptStateLatest   AptState = "latest"
	AptStatePresent  AptState = "present"
	AptStateFixed    AptState = "fixed"
)

// Convert a supported type to an optional (pointer) AptState
func OptionalAptState[T interface {
	*AptState | AptState | *string | string
}](s T) *AptState {
	switch v := any(s).(type) {
	case *AptState:
		return v
	case AptState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := AptState(*v)
		return &val
	case string:
		val := AptState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// If yes or safe, performs an aptitude safe-upgrade.
// If full, performs an aptitude full-upgrade.
// If dist, performs an apt-get dist-upgrade.
// Note: This does not upgrade a specific package, use state=latest for that.
// Note: Since 2.4, apt-get is used as a fall-back if aptitude is not present.
type AptUpgrade string

const (
	AptUpgradeDist AptUpgrade = "dist"
	AptUpgradeFull AptUpgrade = "full"
	AptUpgradeNo   AptUpgrade = "no"
	AptUpgradeSafe AptUpgrade = "safe"
	AptUpgradeYes  AptUpgrade = "yes"
)

// Convert a supported type to an optional (pointer) AptUpgrade
func OptionalAptUpgrade[T interface {
	*AptUpgrade | AptUpgrade | *string | string
}](s T) *AptUpgrade {
	switch v := any(s).(type) {
	case *AptUpgrade:
		return v
	case AptUpgrade:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := AptUpgrade(*v)
		return &val
	case string:
		val := AptUpgrade(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `apt` Ansible module.
type AptParameters struct {
	// A list of package names, like `foo`, or package specifier with version, like
	// `foo=1.0` or `foo>=1.0`. Name wildcards (fnmatch) like `apt*` and version
	// wildcards like `foo=1.0*` are also supported.
	// Do not use single or double quotes around the version when referring to the
	// package name with a specific version, such as `foo=1.0` or `foo>=1.0`.
	Name *[]string `json:"name,omitempty"`

	// Indicates the desired package state. `latest` ensures that the latest
	// version is installed. `build-dep` ensures the package build dependencies are
	// installed. `fixed` attempt to correct a system with broken dependencies in
	// place.
	// default: AptStatePresent
	State *AptState `json:"state,omitempty"`

	// Run the equivalent of `apt-get update` before the operation. Can be run as
	// part of the package installation or as a separate step.
	// Default is not to update the cache.
	UpdateCache *bool `json:"update_cache,omitempty"`

	// Amount of retries if the cache update fails. Also see
	// `update_cache_retry_max_delay`.
	// default: 5
	UpdateCacheRetries *int `json:"update_cache_retries,omitempty"`

	// Use an exponential backoff delay for each retry (see `update_cache_retries`)
	// up to this max delay in seconds.
	// default: 12
	UpdateCacheRetryMaxDelay *int `json:"update_cache_retry_max_delay,omitempty"`

	// Update the apt cache if it is older than the `cache_valid_time`. This option
	// is set in seconds.
	// As of Ansible 2.4, if explicitly set, this sets `update_cache=yes`.
	// default: 0
	CacheValidTime *int `json:"cache_valid_time,omitempty"`

	// Will force purging of configuration files if `state=absent` or
	// `autoremove=yes`.
	// default: "no"
	Purge *bool `json:"purge,omitempty"`

	// Corresponds to the `-t` option for `apt` and sets pin priorities.
	DefaultRelease *string `json:"default_release,omitempty"`

	// Corresponds to the `--no-install-recommends` option for `apt`. `true`
	// installs recommended packages. `false` does not install recommended
	// packages. By default, Ansible will use the same defaults as the operating
	// system. Suggested packages are never installed.
	InstallRecommends *bool `json:"install_recommends,omitempty"`

	// Corresponds to the `--force-yes` to `apt-get` and implies
	// `allow_unauthenticated=yes` and `allow_downgrade=yes`.
	// This option will disable checking both the packages' signatures and the
	// certificates of the web servers they are downloaded from.
	// This option *is not* the equivalent of passing the `-f` flag to `apt-get` on
	// the command line.
	// **This is a destructive operation with the potential to destroy your system,
	// and it should almost never be used.** Please also see `man apt-get` for more
	// information.
	// default: "no"
	Force *bool `json:"force,omitempty"`

	// Run the equivalent of `apt-get clean` to clear out the local repository of
	// retrieved package files. It removes everything but the lock file from
	// `/var/cache/apt/archives/` and `/var/cache/apt/archives/partial/`.
	// Can be run as part of the package installation (clean runs before install)
	// or as a separate step.
	// default: "no"
	Clean *bool `json:"clean,omitempty"`

	// Ignore if packages cannot be authenticated. This is useful for bootstrapping
	// environments that manage their own apt-key setup.
	// `allow_unauthenticated` is only supported with `state`: `install`/`present`.
	// default: "no"
	AllowUnauthenticated *bool `json:"allow_unauthenticated,omitempty"`

	// Corresponds to the `--allow-downgrades` option for `apt`.
	// This option enables the named package and version to replace an already
	// installed higher version of that package.
	// Note that setting `allow_downgrade=true` can make this module behave in a
	// non-idempotent way.
	// (The task could end up with a set of packages that does not match the
	// complete list of specified packages to install).
	// `allow_downgrade` is only supported by `apt` and will be ignored if
	// `aptitude` is detected or specified.
	// default: "no"
	AllowDowngrade *bool `json:"allow_downgrade,omitempty"`

	// Allows changing the version of a package which is on the apt hold list.
	// default: "no"
	AllowChangeHeldPackages *bool `json:"allow_change_held_packages,omitempty"`

	// If yes or safe, performs an aptitude safe-upgrade.
	// If full, performs an aptitude full-upgrade.
	// If dist, performs an apt-get dist-upgrade.
	// Note: This does not upgrade a specific package, use state=latest for that.
	// Note: Since 2.4, apt-get is used as a fall-back if aptitude is not present.
	// default: AptUpgradeNo
	Upgrade *AptUpgrade `json:"upgrade,omitempty"`

	// Add `dpkg` options to `apt` command. Defaults to `-o
	// "Dpkg::Options::=--force-confdef" -o "Dpkg::Options::=--force-confold"`.
	// Options should be supplied as comma separated list.
	// default: "force-confdef,force-confold"
	DpkgOptions *string `json:"dpkg_options,omitempty"`

	// Path to a .deb package on the remote machine.
	// If `://` in the path, ansible will attempt to download deb before
	// installing. (Version added 2.1)
	// Requires the `xz-utils` package to extract the control file of the deb
	// package to install.
	Deb *string `json:"deb,omitempty"`

	// If `true`, remove unused dependency packages for all module states except
	// `build-dep`. It can also be used as the only option.
	// Previous to version 2.4, `autoclean` was also an alias for `autoremove`, now
	// it is its own separate command. See documentation for further information.
	// default: "no"
	Autoremove *bool `json:"autoremove,omitempty"`

	// If `true`, cleans the local repository of retrieved package files that can
	// no longer be downloaded.
	// default: "no"
	Autoclean *bool `json:"autoclean,omitempty"`

	// Force the exit code of `/usr/sbin/policy-rc.d`.
	// For example, if `policy_rc_d=101` the installed package will not trigger a
	// service start.
	// If `/usr/sbin/policy-rc.d` already exists, it is backed up and restored
	// after the package installation.
	// If `null`, the `/usr/sbin/policy-rc.d` is not created/changed.
	// default: nil
	PolicyRcD *int `json:"policy_rc_d,omitempty"`

	// Only upgrade a package if it is already installed.
	// default: "no"
	OnlyUpgrade *bool `json:"only_upgrade,omitempty"`

	// Corresponds to the `--no-remove` option for `apt`.
	// If `true`, it is ensured that no packages will be removed or the task will
	// fail.
	// `fail_on_autoremove` is only supported with `state` except `absent`.
	// `fail_on_autoremove` is only supported by `apt` and will be ignored if
	// `aptitude` is detected or specified.
	// default: "no"
	FailOnAutoremove *bool `json:"fail_on_autoremove,omitempty"`

	// Force usage of apt-get instead of aptitude.
	// default: "no"
	ForceAptGet *bool `json:"force_apt_get,omitempty"`

	// How many seconds will this action wait to acquire a lock on the apt db.
	// Sometimes there is a transitory lock and this will retry at least until
	// timeout is hit.
	// default: 60
	LockTimeout *int `json:"lock_timeout,omitempty"`
}

// Wrap the `AptParameters into an `rpc.RPCCall`.
func (p AptParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: AptName,
			Args: args,
		},
	}, nil
}

// Return values for the `apt` Ansible module.
type AptReturn struct {
	AnsibleCommonReturns

	// if the cache was updated or not
	CacheUpdated *bool `json:"cache_updated,omitempty"`

	// time of the last cache update (0 if unknown)
	CacheUpdateTime *int `json:"cache_update_time,omitempty"`

	// output from apt
	Stdout *string `json:"stdout,omitempty"`

	// error output from apt
	Stderr *string `json:"stderr,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `AptReturn`
func AptReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (AptReturn, error) {
	return cast.AnyToJSONT[AptReturn](r.Result.Result)
}
