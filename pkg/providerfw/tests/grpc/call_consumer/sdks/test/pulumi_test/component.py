# coding=utf-8
# *** WARNING: this file was generated by pulumi-language-python. ***
# *** Do not edit by hand unless you're certain you know what you are doing! ***

import copy
import warnings
import pulumi
import pulumi.runtime
from typing import Any, Mapping, Optional, Sequence, Union, overload
from . import _utilities

__all__ = ["ComponentArgs", "Component"]


@pulumi.input_type
class ComponentArgs:
    def __init__(__self__, *, my_input: Optional[pulumi.Input[str]] = None):
        """
        The set of arguments for constructing a Component resource.
        """
        if my_input is not None:
            pulumi.set(__self__, "my_input", my_input)

    @property
    @pulumi.getter(name="myInput")
    def my_input(self) -> Optional[pulumi.Input[str]]:
        return pulumi.get(self, "my_input")

    @my_input.setter
    def my_input(self, value: Optional[pulumi.Input[str]]):
        pulumi.set(self, "my_input", value)


class Component(pulumi.ComponentResource):
    @overload
    def __init__(
        __self__,
        resource_name: str,
        opts: Optional[pulumi.ResourceOptions] = None,
        my_input: Optional[pulumi.Input[str]] = None,
        __props__=None,
    ):
        """
        Create a Component resource with the given unique name, props, and options.
        :param str resource_name: The name of the resource.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        ...

    @overload
    def __init__(
        __self__,
        resource_name: str,
        args: Optional[ComponentArgs] = None,
        opts: Optional[pulumi.ResourceOptions] = None,
    ):
        """
        Create a Component resource with the given unique name, props, and options.
        :param str resource_name: The name of the resource.
        :param ComponentArgs args: The arguments to use to populate this resource's properties.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        ...

    def __init__(__self__, resource_name: str, *args, **kwargs):
        resource_args, opts = _utilities.get_resource_args_opts(
            ComponentArgs, pulumi.ResourceOptions, *args, **kwargs
        )
        if resource_args is not None:
            __self__._internal_init(resource_name, opts, **resource_args.__dict__)
        else:
            __self__._internal_init(resource_name, *args, **kwargs)

    def _internal_init(
        __self__,
        resource_name: str,
        opts: Optional[pulumi.ResourceOptions] = None,
        my_input: Optional[pulumi.Input[str]] = None,
        __props__=None,
    ):
        opts = pulumi.ResourceOptions.merge(
            _utilities.get_resource_opts_defaults(), opts
        )
        if not isinstance(opts, pulumi.ResourceOptions):
            raise TypeError(
                "Expected resource options to be a ResourceOptions instance"
            )
        if opts.id is not None:
            raise ValueError("ComponentResource classes do not support opts.id")
        else:
            if __props__ is not None:
                raise TypeError(
                    "__props__ is only valid when passed in combination with a valid opts.id to get an existing resource"
                )
            __props__ = ComponentArgs.__new__(ComponentArgs)

            __props__.__dict__["my_input"] = my_input
            __props__.__dict__["my_output"] = None
        super(Component, __self__).__init__(
            "test:index:Component", resource_name, __props__, opts, remote=True
        )

    @property
    @pulumi.getter(name="myOutput")
    def my_output(self) -> pulumi.Output[Optional[str]]:
        return pulumi.get(self, "my_output")

    @pulumi.output_type
    class MyMethodResult:
        def __init__(__self__, resp1=None):
            if resp1 and not isinstance(resp1, str):
                raise TypeError("Expected argument 'resp1' to be a str")
            pulumi.set(__self__, "resp1", resp1)

        @property
        @pulumi.getter
        def resp1(self) -> Optional[str]:
            return pulumi.get(self, "resp1")

    def my_method(
        __self__, *, arg1: Optional[pulumi.Input[str]] = None
    ) -> pulumi.Output["Component.MyMethodResult"]:
        __args__ = dict()
        __args__["__self__"] = __self__
        __args__["arg1"] = arg1
        return pulumi.runtime.call(
            "test:index:Component/myMethod",
            __args__,
            res=__self__,
            typ=Component.MyMethodResult,
        )
