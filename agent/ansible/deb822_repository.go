// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Add and remove deb822 formatted repositories in Debian based distributions.
const Deb822RepositoryName = "deb822_repository"

// Which types of packages to look for from a given source; either binary `deb`
// or source code `deb-src`.
type Deb822RepositoryTypes string

const (
	Deb822RepositoryTypesDeb    Deb822RepositoryTypes = "deb"
	Deb822RepositoryTypesDebSrc Deb822RepositoryTypes = "deb-src"
)

// Convert a supported type to an optional (pointer) Deb822RepositoryTypes
func OptionalDeb822RepositoryTypes[T interface {
	*Deb822RepositoryTypes | Deb822RepositoryTypes | *string | string
}](s T) *Deb822RepositoryTypes {
	switch v := any(s).(type) {
	case *Deb822RepositoryTypes:
		return v
	case Deb822RepositoryTypes:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := Deb822RepositoryTypes(*v)
		return &val
	case string:
		val := Deb822RepositoryTypes(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// A source string state.
type Deb822RepositoryState string

const (
	Deb822RepositoryStateAbsent  Deb822RepositoryState = "absent"
	Deb822RepositoryStatePresent Deb822RepositoryState = "present"
)

// Convert a supported type to an optional (pointer) Deb822RepositoryState
func OptionalDeb822RepositoryState[T interface {
	*Deb822RepositoryState | Deb822RepositoryState | *string | string
}](s T) *Deb822RepositoryState {
	switch v := any(s).(type) {
	case *Deb822RepositoryState:
		return v
	case Deb822RepositoryState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := Deb822RepositoryState(*v)
		return &val
	case string:
		val := Deb822RepositoryState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `deb822_repository` Ansible module.
type Deb822RepositoryParameters struct {
	// Allow downgrading a package that was previously authenticated but is no
	// longer authenticated.
	AllowDowngradeToInsecure *bool `json:"allow_downgrade_to_insecure,omitempty"`

	// Allow insecure repositories.
	AllowInsecure *bool `json:"allow_insecure,omitempty"`

	// Allow repositories signed with a key using a weak digest algorithm.
	AllowWeak *bool `json:"allow_weak,omitempty"`

	// Architectures to search within repository.
	Architectures *[]string `json:"architectures,omitempty"`

	// Controls if APT should try to acquire indexes via a URI constructed from a
	// hashsum of the expected file instead of using the well-known stable filename
	// of the index.
	ByHash *bool `json:"by_hash,omitempty"`

	// Controls if APT should consider the machine's time correct and hence perform
	// time related checks, such as verifying that a Release file is not from the
	// future.
	CheckDate *bool `json:"check_date,omitempty"`

	// Controls if APT should try to detect replay attacks.
	CheckValidUntil *bool `json:"check_valid_until,omitempty"`

	// Components specify different sections of one distribution version present in
	// a `Suite`.
	Components *[]string `json:"components,omitempty"`

	// Controls how far from the future a repository may be.
	DateMaxFuture *int `json:"date_max_future,omitempty"`

	// Tells APT whether the source is enabled or not.
	Enabled *bool `json:"enabled,omitempty"`

	// Determines the path to the `InRelease` file, relative to the normal position
	// of an `InRelease` file.
	InreleasePath *string `json:"inrelease_path,omitempty"`

	// Defines which languages information such as translated package descriptions
	// should be downloaded.
	Languages *[]string `json:"languages,omitempty"`

	// Name of the repo. Specifically used for `X-Repolib-Name` and in naming the
	// repository and signing key files.
	Name string `json:"name"`

	// Controls if APT should try to use `PDiffs` to update old indexes instead of
	// downloading the new indexes entirely.
	Pdiffs *bool `json:"pdiffs,omitempty"`

	// Either a URL to a GPG key, absolute path to a keyring file, one or more
	// fingerprints of keys either in the `trusted.gpg` keyring or in the keyrings
	// in the `trusted.gpg.d/` directory, or an ASCII armored GPG public key block.
	SignedBy *string `json:"signed_by,omitempty"`

	// Suite can specify an exact path in relation to the UR`s` provided, in which
	// case the Components: must be omitted and suite must end with a slash (`/`).
	// Alternatively, it may take the form of a distribution version (for example a
	// version codename like `disco` or `artful`). If the suite does not specify a
	// path, at least one component must be present.
	Suites *[]string `json:"suites,omitempty"`

	// Defines which download targets apt will try to acquire from this source.
	Targets *[]string `json:"targets,omitempty"`

	// Decides if a source is considered trusted or if warnings should be raised
	// before, for example packages are installed from this source.
	Trusted *bool `json:"trusted,omitempty"`

	// Which types of packages to look for from a given source; either binary `deb`
	// or source code `deb-src`.
	// default: []Deb822RepositoryTypes{Deb822RepositoryTypesDeb}
	Types *Deb822RepositoryTypes `json:"types,omitempty"`

	// The URIs must specify the base of the Debian distribution archive, from
	// which APT finds the information it needs.
	Uris *[]string `json:"uris,omitempty"`

	// The octal mode for newly created files in `sources.list.d`.
	// default: "0644"
	Mode *any `json:"mode,omitempty"`

	// A source string state.
	// default: Deb822RepositoryStatePresent
	State *Deb822RepositoryState `json:"state,omitempty"`
}

// Wrap the `Deb822RepositoryParameters into an `rpc.RPCCall`.
func (p Deb822RepositoryParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: Deb822RepositoryName,
			Args: args,
		},
	}, nil
}

// Return values for the `deb822_repository` Ansible module.
type Deb822RepositoryReturn struct {
	AnsibleCommonReturns

	// A source string for the repository
	Repo *string `json:"repo,omitempty"`

	// Path to the repository file
	Dest *string `json:"dest,omitempty"`

	// Path to the signed_by key file
	KeyFilename *string `json:"key_filename,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `Deb822RepositoryReturn`
func Deb822RepositoryReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (Deb822RepositoryReturn, error) {
	return cast.AnyToJSONT[Deb822RepositoryReturn](r.Result.Result)
}
