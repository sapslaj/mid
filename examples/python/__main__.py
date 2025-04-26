import pulumi
import pulumi_mid as mid

my_random_resource = mid.Random("myRandomResource", length=24)
pulumi.export("output", {
    "value": my_random_resource.result,
})
