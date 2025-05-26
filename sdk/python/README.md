# mid

Pulumi-native configuration management

> [!CAUTION]
> This provider is PRE-ALPHA software and _NOT_ fit for production use!

mid combines the simplicity of Ansible, the declarativeness of Nix, and the
power of Pulumi to make server configuration management simple, easy, and fast.
mid is a "middle ground" between all of them to give you just what you need
with as minimal overhead as possible.

## Design Principles

### Goals

- Agentless[^1]; do everything over SSH without a persistent agent service
  running or any further networking required.
- Manage machines quickly and in parallel (like Ansible, but with even more
  parallelization).
- Declarative and determinate (e.g. deleting the Pulumi resource should delete
  the associated thing on the server[^2])
- Mostly drop-in replacement for the remote module of the [Pulumi Command Provider](https://www.pulumi.com/registry/packages/command/)

### Non-goals

- Nix-levels of determinism - achieving fully deterministic systems means
  managing the entire world deterministically and that just isn't feasible. If
  you need/want that level of determinism, just use Nix and NixOS.
- Interop with other CM systems (Ansible, Puppet, Salt) - mid incorporates a
  _very_ stripped down fork of Ansible that uses a custom Go-based task execution
  engine. This allows for significantly faster feature development and provides
  valuable escape hatches for the many cases where a first-class resource or
  function hasn't been developed yet. However, any kind of support for interop
  with official Ansible or any other CM systems is not planned[^3].
  [pulumi-ansible-provisioner](https://github.com/sapslaj/pulumi-ansible-provisioner/)
  might be of interest for using existing Ansible roles.
- Pull-based or agent-based workflows - I'm hyperfocusing on push-based
  deployments to keep it simple. Check out my
  [pulumi-aws-ec2-instance](https://github.com/sapslaj/pulumi-aws-ec2-instance)
  component resources for ideas on how to interop Ansible with EC2 userdata (or
  your cloud's equivalent).
- Windows support - Only Linux is supported for now. Other unix-likes are
  future goals.

### Future-goals

- Better OS support (except Windows) - Right now only Linux is supported, and
  really only Ubuntu/Debian are _truly_ supported[^4]. While I expect
  Debian-based distros to have the most support, there is no reason that other
  distros can't be supported just as well. Other unix-likes such as the BSDs
  should be better supported as well since those are plenty prevalent albeit not
  to the same degree as Linux. macOS support might happen as a side effect of
  other work but I'm unsure if having any kind of first-class support is useful.
- Be usable by non-root (and non-sudo) - The vast majority of use cases require
  root and a fast majority of systems have sudo. Right now this is hardcoded[^5] but
  in the future it would be nice to have this be more flexible.
- Pluggable module system - Right now everything baked into the provider but it
  might be nice to be able to expand it somehow.
- More language support - Only Go, TypeScript, Python, and YAML are supported
  at the moment since those are the only languages I use with Pulumi. C# and Java
  support will come eventually.

[^1]: _Technically_ there is an "agent" that runs on the remote node, however
    it only runs for the duration that the Pulumi provider runs and
    communicates with the provider over stdin/stdout, not TCP or any other side
    channel network protocol. This is very similar to how [Ansible modules are executed](https://docs.ansible.com/ansible/latest/dev_guide/developing_program_flow_modules.html#how-modules-are-executed).
    So if you are okay with using Ansible, then you should be okay with using mid.

[^2]: There will be cases where mid is unable to know what to do on a delete.
    It should _generally_ try to do the "correct" thing but this is not
    guaranteed. If you want to err on the side of caution, use Pulumi's
    [`retainOnDelete`](https://www.pulumi.com/docs/iac/concepts/options/retainondelete/)
    resource option to skip doing any delete operations.

[^3]: While "first-class" support for external CM system interop is not
    planned, a byproduct of the embedded Ansible module system means there is
    support for gathering facts by using the `ansibleExecute` function with the
    `setup` module. There is even support for running the `facter` module to get
    facts from [Facter](https://www.puppet.com/docs/puppet/7/facter.html). I have
    no intention of removing the ability to do this, since this is extremely useful
    in certain cases. Just don't expect any kind of top-level `mid.getFacts()`
    function.

[^4]: Ubuntu is the most supported distro 1. because it is what I use in my
    servers (for better or worse) and 2. because it by far has the best
    cloud-init integration which makes initial bootstrapping and testing so much
    easier. That said, on my non-servers I use Arch (btw), so any OS or distro that
    can run Pulumi should be able to use the mid provider, it just might not be
    able to be configured by mid.

[^5]: It will use sudo if present. If the `sudo` command isn't found it will
    run any commands without sudo. There is a high likelihood those commands
    will fail unless running as `root` though. Also, there is no support for sudo
    passwords for now, but that is planned. cloud-init sets `NOPASSWD` for the
    default user by default, and using the default user is the main way I
    envisioned this being used, hence why I haven't prioritized this.

## Installation and usage

> [!CAUTION]
> Again, this is PRE-ALPHA software. It is not fit for general use!

### Go

Grab the module

```shell
go get github.com/sapslaj/mid/sdk
```

Set up a provider instance

```go
// main.go
package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/mid/sdk/go/mid"
	"github.com/sapslaj/mid/sdk/go/mid/resource"
	"github.com/sapslaj/mid/sdk/go/mid/types"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		provider, err := mid.NewProvider(ctx, "provider", &mid.ProviderArgs{
			Connection: &types.ConnectionArgs{
				// TODO: use ESC for this for something
				User:     pulumi.String("root"),
				Password: pulumi.String("hunter2"),
				Host:     pulumi.String("localhost"),
				Port:     pulumi.Float64(22),
			},
		})
		if err != nil {
			return err
		}
		return nil
	})
}
```

Add some resources

```go
// main.go

// ...
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// ...

		_, err = resource.NewPackage(ctx, "vim", &resource.PackageArgs{
			Name:   pulumi.String("vim"),
			Ensure: pulumi.String("present"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		return nil
	})
}
```

Go!

```shell
pulumi up
```

### TypeScript

mid is not published on NPM yet. [Set up GitHub Packages as an NPM registry for
the `@sapslaj`
scope](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-npm-registry#installing-a-package)
first.

Then npm install

```shell
npm install --save @sapslaj/pulumi-mid
```

Set up a provider instance

```typescript
// index.ts
import * as pulumi from "@pulumi/pulumi";
import * as mid from "@sapslaj/pulumi-mid";

const provider = new mid.Provider("provider", {
  connection: {
    // TODO: use ESC for this or something
    user: "root",
    password: "hunter2",
    host: "localhost",
    port: 22,
  },
});
```

Add some resources

```typescript
// ...
new mid.resource.Package("vim", {
  name: "vim",
  ensure: "present",
}, {
  provider: provider,
});
```

Go!

```shell
pulumi up
```

### Python

mid is not published on PyPI yet. Install the Pulumi provider package directly
from GitHub. Note that you might want to replace `@main` with a release git
tag.

```shell
pip install 'pulumi_mid @ git+https://github.com/sapslaj/mid.git@main#subdirectory=sdk/python'
```

Set up a provider instance

```python
# __main__.py
import pulumi
import pulumi_mid as mid

provider = mid.Provider(
    "provider",
    # TODO: use ESC for this or something
    connection={
        "user": "root",
        "password": "hunter2",
        "host": "localhost",
        "port": 22,
    },
)
```

Add some resources

```python
vim = mid.resource.Package(
    "vim",
    name="vim",
    ensure="present",
    opts=pulumi.ResourceOptions(provider=provider),
)
```

Go!

```shell
pulumi up
```
