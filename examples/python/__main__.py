import pulumi
import pulumi_mid as mid

provider = mid.Provider(
    "provider",
    connection={
        "user": "root",
        "password": "hunter2",
        "host": "localhost",
        "port": 2222,
    },
)
vim = mid.resource.Package("vim", opts=pulumi.ResourceOptions(provider=provider))
emacs = mid.resource.Package(
    "emacs",
    name="emacs",
    ensure="absent",
    opts=pulumi.ResourceOptions(provider=provider),
)
