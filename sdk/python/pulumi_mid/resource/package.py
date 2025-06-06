# coding=utf-8
# *** WARNING: this file was generated by pulumi-language-python. ***
# *** Do not edit by hand unless you're certain you know what you are doing! ***

import builtins
import copy
import warnings
import sys
import pulumi
import pulumi.runtime
from typing import Any, Mapping, Optional, Sequence, Union, overload

if sys.version_info >= (3, 11):
    from typing import NotRequired, TypedDict, TypeAlias
else:
    from typing_extensions import NotRequired, TypedDict, TypeAlias
from .. import _utilities
from .. import types as _types

__all__ = ["PackageArgs", "Package"]


@pulumi.input_type
class PackageArgs:
    def __init__(
        __self__,
        *,
        ensure: Optional[pulumi.Input[builtins.str]] = None,
        name: Optional[pulumi.Input[builtins.str]] = None,
        names: Optional[pulumi.Input[Sequence[pulumi.Input[builtins.str]]]] = None,
        triggers: Optional[pulumi.Input["_types.TriggersInputArgs"]] = None,
    ):
        """
        The set of arguments for constructing a Package resource.
        """
        if ensure is not None:
            pulumi.set(__self__, "ensure", ensure)
        if name is not None:
            pulumi.set(__self__, "name", name)
        if names is not None:
            pulumi.set(__self__, "names", names)
        if triggers is not None:
            pulumi.set(__self__, "triggers", triggers)

    @property
    @pulumi.getter
    def ensure(self) -> Optional[pulumi.Input[builtins.str]]:
        return pulumi.get(self, "ensure")

    @ensure.setter
    def ensure(self, value: Optional[pulumi.Input[builtins.str]]):
        pulumi.set(self, "ensure", value)

    @property
    @pulumi.getter
    def name(self) -> Optional[pulumi.Input[builtins.str]]:
        return pulumi.get(self, "name")

    @name.setter
    def name(self, value: Optional[pulumi.Input[builtins.str]]):
        pulumi.set(self, "name", value)

    @property
    @pulumi.getter
    def names(self) -> Optional[pulumi.Input[Sequence[pulumi.Input[builtins.str]]]]:
        return pulumi.get(self, "names")

    @names.setter
    def names(
        self, value: Optional[pulumi.Input[Sequence[pulumi.Input[builtins.str]]]]
    ):
        pulumi.set(self, "names", value)

    @property
    @pulumi.getter
    def triggers(self) -> Optional[pulumi.Input["_types.TriggersInputArgs"]]:
        return pulumi.get(self, "triggers")

    @triggers.setter
    def triggers(self, value: Optional[pulumi.Input["_types.TriggersInputArgs"]]):
        pulumi.set(self, "triggers", value)


@pulumi.type_token("mid:resource:Package")
class Package(pulumi.CustomResource):
    @overload
    def __init__(
        __self__,
        resource_name: str,
        opts: Optional[pulumi.ResourceOptions] = None,
        ensure: Optional[pulumi.Input[builtins.str]] = None,
        name: Optional[pulumi.Input[builtins.str]] = None,
        names: Optional[pulumi.Input[Sequence[pulumi.Input[builtins.str]]]] = None,
        triggers: Optional[
            pulumi.Input[
                Union["_types.TriggersInputArgs", "_types.TriggersInputArgsDict"]
            ]
        ] = None,
        __props__=None,
    ):
        """
        Create a Package resource with the given unique name, props, and options.
        :param str resource_name: The name of the resource.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        ...

    @overload
    def __init__(
        __self__,
        resource_name: str,
        args: Optional[PackageArgs] = None,
        opts: Optional[pulumi.ResourceOptions] = None,
    ):
        """
        Create a Package resource with the given unique name, props, and options.
        :param str resource_name: The name of the resource.
        :param PackageArgs args: The arguments to use to populate this resource's properties.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        ...

    def __init__(__self__, resource_name: str, *args, **kwargs):
        resource_args, opts = _utilities.get_resource_args_opts(
            PackageArgs, pulumi.ResourceOptions, *args, **kwargs
        )
        if resource_args is not None:
            __self__._internal_init(resource_name, opts, **resource_args.__dict__)
        else:
            __self__._internal_init(resource_name, *args, **kwargs)

    def _internal_init(
        __self__,
        resource_name: str,
        opts: Optional[pulumi.ResourceOptions] = None,
        ensure: Optional[pulumi.Input[builtins.str]] = None,
        name: Optional[pulumi.Input[builtins.str]] = None,
        names: Optional[pulumi.Input[Sequence[pulumi.Input[builtins.str]]]] = None,
        triggers: Optional[
            pulumi.Input[
                Union["_types.TriggersInputArgs", "_types.TriggersInputArgsDict"]
            ]
        ] = None,
        __props__=None,
    ):
        opts = pulumi.ResourceOptions.merge(
            _utilities.get_resource_opts_defaults(), opts
        )
        if not isinstance(opts, pulumi.ResourceOptions):
            raise TypeError(
                "Expected resource options to be a ResourceOptions instance"
            )
        if opts.id is None:
            if __props__ is not None:
                raise TypeError(
                    "__props__ is only valid when passed in combination with a valid opts.id to get an existing resource"
                )
            __props__ = PackageArgs.__new__(PackageArgs)

            __props__.__dict__["ensure"] = ensure
            __props__.__dict__["name"] = name
            __props__.__dict__["names"] = names
            __props__.__dict__["triggers"] = triggers
        super(Package, __self__).__init__(
            "mid:resource:Package", resource_name, __props__, opts
        )

    @staticmethod
    def get(
        resource_name: str,
        id: pulumi.Input[str],
        opts: Optional[pulumi.ResourceOptions] = None,
    ) -> "Package":
        """
        Get an existing Package resource's state with the given name, id, and optional extra
        properties used to qualify the lookup.

        :param str resource_name: The unique name of the resulting resource.
        :param pulumi.Input[str] id: The unique provider ID of the resource to lookup.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        opts = pulumi.ResourceOptions.merge(opts, pulumi.ResourceOptions(id=id))

        __props__ = PackageArgs.__new__(PackageArgs)

        __props__.__dict__["ensure"] = None
        __props__.__dict__["name"] = None
        __props__.__dict__["names"] = None
        __props__.__dict__["triggers"] = None
        return Package(resource_name, opts=opts, __props__=__props__)

    @property
    @pulumi.getter
    def ensure(self) -> pulumi.Output[builtins.str]:
        return pulumi.get(self, "ensure")

    @property
    @pulumi.getter
    def name(self) -> pulumi.Output[Optional[builtins.str]]:
        return pulumi.get(self, "name")

    @property
    @pulumi.getter
    def names(self) -> pulumi.Output[Optional[Sequence[builtins.str]]]:
        return pulumi.get(self, "names")

    @property
    @pulumi.getter
    def triggers(self) -> pulumi.Output["_types.outputs.TriggersOutput"]:
        return pulumi.get(self, "triggers")
