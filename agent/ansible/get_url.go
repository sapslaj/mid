// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Downloads files from HTTP, HTTPS, or FTP to the remote server. The remote
// server `must` have direct access to the remote resource.
// By default, if an environment variable `<protocol>_proxy` is set on the
// target host, requests will be sent through that proxy. This behaviour can be
// overridden by setting a variable for this task (see `setting the
// environment,playbooks_environment`), or by using the use_proxy option.
// HTTP redirects can redirect from HTTP to HTTPS so you should be sure that
// your proxy environment for both protocols is correct.
// From Ansible 2.4 when run with `--check`, it will do a HEAD request to
// validate the URL but will not download the entire file or verify it against
// hashes and will report incorrect changed status.
// For Windows targets, use the `ansible.windows.win_get_url` module instead.
const GetUrlName = "get_url"

// Parameters for the `get_url` Ansible module.
type GetUrlParameters struct {
	// SSL/TLS Ciphers to use for the request.
	// When a list is provided, all ciphers are joined in order with `:`.
	// See the `OpenSSL Cipher List
	// Format,https://www.openssl.org/docs/manmaster/man1/openssl-
	// ciphers.html#CIPHER-LIST-FORMAT` for more details.
	// The available ciphers is dependent on the Python and OpenSSL/LibreSSL
	// versions.
	Ciphers *[]string `json:"ciphers,omitempty"`

	// Whether to attempt to decompress gzip content-encoded responses.
	// default: true
	Decompress *bool `json:"decompress,omitempty"`

	// HTTP, HTTPS, or FTP URL in the form
	// `(http|https|ftp`://[user[:pass]]@host.domain[:port]/path).
	Url string `json:"url"`

	// Absolute path of where to download the file to.
	// If `dest` is a directory, either the server provided filename or, if none
	// provided, the base name of the URL on the remote server will be used. If a
	// directory, `force` has no effect.
	// If `dest` is a directory, the file will always be downloaded (regardless of
	// the `force` and `checksum` option), but replaced only if the contents
	// changed.
	Dest string `json:"dest"`

	// Absolute path of where temporary file is downloaded to.
	// When run on Ansible 2.5 or greater, path defaults to ansible's `remote_tmp`
	// setting.
	// When run on Ansible prior to 2.5, it defaults to `TMPDIR`, `TEMP` or `TMP`
	// env variables or a platform specific value.
	// `https://docs.python.org/3/library/tempfile.html#tempfile.tempdir`.
	TmpDest *string `json:"tmp_dest,omitempty"`

	// If `true` and `dest` is not a directory, will download the file every time
	// and replace the file if the contents change. If `false`, the file will only
	// be downloaded if the destination does not exist. Generally should be `true`
	// only for small local files.
	// Prior to 0.6, this module behaved as if `true` was the default.
	// default: false
	Force *bool `json:"force,omitempty"`

	// Create a backup file including the timestamp information so you can get the
	// original file back if you somehow clobbered it incorrectly.
	// default: false
	Backup *bool `json:"backup,omitempty"`

	// If a checksum is passed to this parameter, the digest of the destination
	// file will be calculated after it is downloaded to ensure its integrity and
	// verify that the transfer completed successfully. Format:
	// <algorithm>:<checksum|url>, for example
	// `checksum="sha256:D98291AC[...]B6DC7B97",
	// C(checksum="sha256:http://example.com/path/sha256sum.txt"`.
	// If you worry about portability, only the sha1 algorithm is available on all
	// platforms and python versions.
	// The Python `hashlib` module is responsible for providing the available
	// algorithms. The choices vary based on Python version and OpenSSL version.
	// On systems running in FIPS compliant mode, the `md5` algorithm may be
	// unavailable.
	// Additionally, if a checksum is passed to this parameter, and the file exist
	// under the `dest` location, the `destination_checksum` would be calculated,
	// and if checksum equals `destination_checksum`, the file download would be
	// skipped (unless `force=true`). If the checksum does not equal
	// `destination_checksum`, the destination file is deleted.
	// If the checksum URL requires username and password, `url_username` and
	// `url_password` are used to download the checksum file.
	// default: ""
	Checksum *string `json:"checksum,omitempty"`

	// if `false`, it will not use a proxy, even if one is defined in an
	// environment variable on the target hosts.
	// default: true
	UseProxy *bool `json:"use_proxy,omitempty"`

	// If `false`, SSL certificates will not be validated.
	// This should only be used on personally controlled sites using self-signed
	// certificates.
	// default: true
	ValidateCerts *bool `json:"validate_certs,omitempty"`

	// Timeout in seconds for URL request.
	// default: 10
	Timeout *int `json:"timeout,omitempty"`

	// Add custom HTTP headers to a request in hash/dict format.
	// The hash/dict format was added in Ansible 2.6.
	// Previous versions used a `"key:value,key:value"` string format.
	// The `"key:value,key:value"` string format is deprecated and has been removed
	// in version 2.10.
	Headers *map[string]any `json:"headers,omitempty"`

	// The username for use in HTTP basic authentication.
	// This parameter can be used without `url_password` for sites that allow empty
	// passwords.
	// Since version 2.8 you can also use the `username` alias for this option.
	UrlUsername *string `json:"url_username,omitempty"`

	// The password for use in HTTP basic authentication.
	// If the `url_username` parameter is not specified, the `url_password`
	// parameter will not be used.
	// Since version 2.8 you can also use the `password` alias for this option.
	UrlPassword *string `json:"url_password,omitempty"`

	// Force the sending of the Basic authentication header upon initial request.
	// httplib2, the library used by the uri module only sends authentication
	// information when a webservice responds to an initial request with a 401
	// status. Since some basic auth services do not properly send a 401, logins
	// will fail.
	// default: false
	ForceBasicAuth *bool `json:"force_basic_auth,omitempty"`

	// PEM formatted certificate chain file to be used for SSL client
	// authentication.
	// This file can also include the key as well, and if the key is included,
	// `client_key` is not required.
	ClientCert *string `json:"client_cert,omitempty"`

	// PEM formatted file that contains your private key to be used for SSL client
	// authentication.
	// If `client_cert` contains both the certificate and key, this option is not
	// required.
	ClientKey *string `json:"client_key,omitempty"`

	// Header to identify as, generally appears in web server logs.
	// default: "ansible-httpget"
	HttpAgent *string `json:"http_agent,omitempty"`

	// A list of header names that will not be sent on subsequent redirected
	// requests. This list is case insensitive. By default all headers will be
	// redirected. In some cases it may be beneficial to list headers such as
	// `Authorization` here to avoid potential credential exposure.
	// default: []
	UnredirectedHeaders *[]string `json:"unredirected_headers,omitempty"`

	// Use GSSAPI to perform the authentication, typically this is for Kerberos or
	// Kerberos through Negotiate authentication.
	// Requires the Python library `gssapi,https://github.com/pythongssapi/python-
	// gssapi` to be installed.
	// Credentials for GSSAPI can be specified with `url_username`/`url_password`
	// or with the GSSAPI env var `KRB5CCNAME` that specified a custom Kerberos
	// credential cache.
	// NTLM authentication is `not` supported even if the GSSAPI mech for NTLM has
	// been installed.
	// default: false
	UseGssapi *bool `json:"use_gssapi,omitempty"`

	// Determining whether to use credentials from `~/.netrc` file.
	// By default `.netrc` is used with Basic authentication headers.
	// When `false`, `.netrc` credentials are ignored.
	// default: true
	UseNetrc *bool `json:"use_netrc,omitempty"`

	// The permissions the resulting filesystem object should have.
	// For those used to `/usr/bin/chmod` remember that modes are actually octal
	// numbers. You must give Ansible enough information to parse them correctly.
	// For consistent results, quote octal numbers (for example, `'644'` or
	// `'1777'`) so Ansible receives a string and can do its own conversion from
	// string into number. Adding a leading zero (for example, `0755`) works
	// sometimes, but can fail in loops and some other circumstances.
	// Giving Ansible a number without following either of these rules will end up
	// with a decimal number which will have unexpected results.
	// As of Ansible 1.8, the mode may be specified as a symbolic mode (for
	// example, `u+rwx` or `u=rw,g=r,o=r`).
	// If `mode` is not specified and the destination filesystem object `does not`
	// exist, the default `umask` on the system will be used when setting the mode
	// for the newly created filesystem object.
	// If `mode` is not specified and the destination filesystem object `does`
	// exist, the mode of the existing filesystem object will be used.
	// Specifying `mode` is the best way to ensure filesystem objects are created
	// with the correct permissions. See CVE-2020-1736 for further details.
	Mode *any `json:"mode,omitempty"`

	// Name of the user that should own the filesystem object, as would be fed to
	// `chown`.
	// When left unspecified, it uses the current user unless you are root, in
	// which case it can preserve the previous ownership.
	// Specifying a numeric username will be assumed to be a user ID and not a
	// username. Avoid numeric usernames to avoid this confusion.
	Owner *string `json:"owner,omitempty"`

	// Name of the group that should own the filesystem object, as would be fed to
	// `chown`.
	// When left unspecified, it uses the current group of the current user unless
	// you are root, in which case it can preserve the previous ownership.
	Group *string `json:"group,omitempty"`

	// The user part of the SELinux filesystem object context.
	// By default it uses the `system` policy, where applicable.
	// When set to `_default`, it will use the `user` portion of the policy if
	// available.
	Seuser *string `json:"seuser,omitempty"`

	// The role part of the SELinux filesystem object context.
	// When set to `_default`, it will use the `role` portion of the policy if
	// available.
	Serole *string `json:"serole,omitempty"`

	// The type part of the SELinux filesystem object context.
	// When set to `_default`, it will use the `type` portion of the policy if
	// available.
	Setype *string `json:"setype,omitempty"`

	// The level part of the SELinux filesystem object context.
	// This is the MLS/MCS attribute, sometimes known as the `range`.
	// When set to `_default`, it will use the `level` portion of the policy if
	// available.
	Selevel *string `json:"selevel,omitempty"`

	// Influence when to use atomic operation to prevent data corruption or
	// inconsistent reads from the target filesystem object.
	// By default this module uses atomic operations to prevent data corruption or
	// inconsistent reads from the target filesystem objects, but sometimes systems
	// are configured or just broken in ways that prevent this. One example is
	// docker mounted filesystem objects, which cannot be updated atomically from
	// inside the container and can only be written in an unsafe manner.
	// This option allows Ansible to fall back to unsafe methods of updating
	// filesystem objects when atomic operations fail (however, it doesn't force
	// Ansible to perform unsafe writes).
	// IMPORTANT! Unsafe writes are subject to race conditions and can lead to data
	// corruption.
	// default: false
	UnsafeWrites *bool `json:"unsafe_writes,omitempty"`

	// The attributes the resulting filesystem object should have.
	// To get supported flags look at the man page for `chattr` on the target
	// system.
	// This string should contain the attributes in the same order as the one
	// displayed by `lsattr`.
	// The `=` operator is assumed as default, otherwise `+` or `-` operators need
	// to be included in the string.
	Attributes *string `json:"attributes,omitempty"`
}

// Wrap the `GetUrlParameters into an `rpc.RPCCall`.
func (p GetUrlParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: GetUrlName,
			Args: args,
		},
	}, nil
}

// Return values for the `get_url` Ansible module.
type GetUrlReturn struct {
	AnsibleCommonReturns

	// name of backup file created after download
	BackupFile *string `json:"backup_file,omitempty"`

	// sha1 checksum of the file after copy
	ChecksumDest *string `json:"checksum_dest,omitempty"`

	// sha1 checksum of the file
	ChecksumSrc *string `json:"checksum_src,omitempty"`

	// destination file/path
	Dest *string `json:"dest,omitempty"`

	// The number of seconds that elapsed while performing the download
	Elapsed *int `json:"elapsed,omitempty"`

	// group id of the file
	Gid *int `json:"gid,omitempty"`

	// group of the file
	Group *string `json:"group,omitempty"`

	// md5 checksum of the file after download
	Md5sum *string `json:"md5sum,omitempty"`

	// permissions of the target
	Mode *string `json:"mode,omitempty"`

	// the HTTP message from the request
	Msg *string `json:"msg,omitempty"`

	// owner of the file
	Owner *string `json:"owner,omitempty"`

	// the SELinux security context of the file
	Secontext *string `json:"secontext,omitempty"`

	// size of the target
	Size *int `json:"size,omitempty"`

	// source file used after download
	Src *string `json:"src,omitempty"`

	// state of the target
	State *string `json:"state,omitempty"`

	// the HTTP status code from the request
	StatusCode *int `json:"status_code,omitempty"`

	// owner id of the file, after execution
	Uid *int `json:"uid,omitempty"`

	// the actual URL used for the request
	Url *string `json:"url,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `GetUrlReturn`
func GetUrlReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (GetUrlReturn, error) {
	return cast.AnyToJSONT[GetUrlReturn](r.Result.Result)
}
