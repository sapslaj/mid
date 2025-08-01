# coding=utf-8
# *** WARNING: this file was generated by pulumi-language-python. ***
# *** Do not edit by hand unless you're certain you know what you are doing! ***

import builtins as _builtins
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
from .. import _inputs as _root_inputs
from .. import outputs as _root_outputs

__all__ = ["ExecArgs", "Exec"]


@pulumi.input_type
class ExecArgs:
    def __init__(
        __self__,
        *,
        create: pulumi.Input["_root_inputs.ExecCommandArgs"],
        config: Optional[pulumi.Input["_root_inputs.ResourceConfigArgs"]] = None,
        connection: Optional[pulumi.Input["_root_inputs.ConnectionArgs"]] = None,
        delete: Optional[pulumi.Input["_root_inputs.ExecCommandArgs"]] = None,
        delete_before_replace: Optional[pulumi.Input[_builtins.bool]] = None,
        dir: Optional[pulumi.Input[_builtins.str]] = None,
        environment: Optional[
            pulumi.Input[Mapping[str, pulumi.Input[_builtins.str]]]
        ] = None,
        expand_argument_vars: Optional[pulumi.Input[_builtins.bool]] = None,
        logging: Optional[pulumi.Input[_builtins.str]] = None,
        triggers: Optional[pulumi.Input["_root_inputs.TriggersInputArgs"]] = None,
        update: Optional[pulumi.Input["_root_inputs.ExecCommandArgs"]] = None,
    ):
        """
        The set of arguments for constructing a Exec resource.
        """
        pulumi.set(__self__, "create", create)
        if config is not None:
            pulumi.set(__self__, "config", config)
        if connection is not None:
            pulumi.set(__self__, "connection", connection)
        if delete is not None:
            pulumi.set(__self__, "delete", delete)
        if delete_before_replace is not None:
            pulumi.set(__self__, "delete_before_replace", delete_before_replace)
        if dir is not None:
            pulumi.set(__self__, "dir", dir)
        if environment is not None:
            pulumi.set(__self__, "environment", environment)
        if expand_argument_vars is not None:
            pulumi.set(__self__, "expand_argument_vars", expand_argument_vars)
        if logging is not None:
            pulumi.set(__self__, "logging", logging)
        if triggers is not None:
            pulumi.set(__self__, "triggers", triggers)
        if update is not None:
            pulumi.set(__self__, "update", update)

    @_builtins.property
    @pulumi.getter
    def create(self) -> pulumi.Input["_root_inputs.ExecCommandArgs"]:
        return pulumi.get(self, "create")

    @create.setter
    def create(self, value: pulumi.Input["_root_inputs.ExecCommandArgs"]):
        pulumi.set(self, "create", value)

    @_builtins.property
    @pulumi.getter
    def config(self) -> Optional[pulumi.Input["_root_inputs.ResourceConfigArgs"]]:
        return pulumi.get(self, "config")

    @config.setter
    def config(self, value: Optional[pulumi.Input["_root_inputs.ResourceConfigArgs"]]):
        pulumi.set(self, "config", value)

    @_builtins.property
    @pulumi.getter
    def connection(self) -> Optional[pulumi.Input["_root_inputs.ConnectionArgs"]]:
        return pulumi.get(self, "connection")

    @connection.setter
    def connection(self, value: Optional[pulumi.Input["_root_inputs.ConnectionArgs"]]):
        pulumi.set(self, "connection", value)

    @_builtins.property
    @pulumi.getter
    def delete(self) -> Optional[pulumi.Input["_root_inputs.ExecCommandArgs"]]:
        return pulumi.get(self, "delete")

    @delete.setter
    def delete(self, value: Optional[pulumi.Input["_root_inputs.ExecCommandArgs"]]):
        pulumi.set(self, "delete", value)

    @_builtins.property
    @pulumi.getter(name="deleteBeforeReplace")
    def delete_before_replace(self) -> Optional[pulumi.Input[_builtins.bool]]:
        return pulumi.get(self, "delete_before_replace")

    @delete_before_replace.setter
    def delete_before_replace(self, value: Optional[pulumi.Input[_builtins.bool]]):
        pulumi.set(self, "delete_before_replace", value)

    @_builtins.property
    @pulumi.getter
    def dir(self) -> Optional[pulumi.Input[_builtins.str]]:
        return pulumi.get(self, "dir")

    @dir.setter
    def dir(self, value: Optional[pulumi.Input[_builtins.str]]):
        pulumi.set(self, "dir", value)

    @_builtins.property
    @pulumi.getter
    def environment(
        self,
    ) -> Optional[pulumi.Input[Mapping[str, pulumi.Input[_builtins.str]]]]:
        return pulumi.get(self, "environment")

    @environment.setter
    def environment(
        self, value: Optional[pulumi.Input[Mapping[str, pulumi.Input[_builtins.str]]]]
    ):
        pulumi.set(self, "environment", value)

    @_builtins.property
    @pulumi.getter(name="expandArgumentVars")
    def expand_argument_vars(self) -> Optional[pulumi.Input[_builtins.bool]]:
        return pulumi.get(self, "expand_argument_vars")

    @expand_argument_vars.setter
    def expand_argument_vars(self, value: Optional[pulumi.Input[_builtins.bool]]):
        pulumi.set(self, "expand_argument_vars", value)

    @_builtins.property
    @pulumi.getter
    def logging(self) -> Optional[pulumi.Input[_builtins.str]]:
        return pulumi.get(self, "logging")

    @logging.setter
    def logging(self, value: Optional[pulumi.Input[_builtins.str]]):
        pulumi.set(self, "logging", value)

    @_builtins.property
    @pulumi.getter
    def triggers(self) -> Optional[pulumi.Input["_root_inputs.TriggersInputArgs"]]:
        return pulumi.get(self, "triggers")

    @triggers.setter
    def triggers(self, value: Optional[pulumi.Input["_root_inputs.TriggersInputArgs"]]):
        pulumi.set(self, "triggers", value)

    @_builtins.property
    @pulumi.getter
    def update(self) -> Optional[pulumi.Input["_root_inputs.ExecCommandArgs"]]:
        return pulumi.get(self, "update")

    @update.setter
    def update(self, value: Optional[pulumi.Input["_root_inputs.ExecCommandArgs"]]):
        pulumi.set(self, "update", value)


@pulumi.type_token("mid:resource:Exec")
class Exec(pulumi.CustomResource):
    @overload
    def __init__(
        __self__,
        resource_name: str,
        opts: Optional[pulumi.ResourceOptions] = None,
        config: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ResourceConfigArgs",
                    "_root_inputs.ResourceConfigArgsDict",
                ]
            ]
        ] = None,
        connection: Optional[
            pulumi.Input[
                Union["_root_inputs.ConnectionArgs", "_root_inputs.ConnectionArgsDict"]
            ]
        ] = None,
        create: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ExecCommandArgs", "_root_inputs.ExecCommandArgsDict"
                ]
            ]
        ] = None,
        delete: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ExecCommandArgs", "_root_inputs.ExecCommandArgsDict"
                ]
            ]
        ] = None,
        delete_before_replace: Optional[pulumi.Input[_builtins.bool]] = None,
        dir: Optional[pulumi.Input[_builtins.str]] = None,
        environment: Optional[
            pulumi.Input[Mapping[str, pulumi.Input[_builtins.str]]]
        ] = None,
        expand_argument_vars: Optional[pulumi.Input[_builtins.bool]] = None,
        logging: Optional[pulumi.Input[_builtins.str]] = None,
        triggers: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.TriggersInputArgs",
                    "_root_inputs.TriggersInputArgsDict",
                ]
            ]
        ] = None,
        update: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ExecCommandArgs", "_root_inputs.ExecCommandArgsDict"
                ]
            ]
        ] = None,
        __props__=None,
    ):
        """
        Create a Exec resource with the given unique name, props, and options.
        :param str resource_name: The name of the resource.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        ...

    @overload
    def __init__(
        __self__,
        resource_name: str,
        args: ExecArgs,
        opts: Optional[pulumi.ResourceOptions] = None,
    ):
        """
        Create a Exec resource with the given unique name, props, and options.
        :param str resource_name: The name of the resource.
        :param ExecArgs args: The arguments to use to populate this resource's properties.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        ...

    def __init__(__self__, resource_name: str, *args, **kwargs):
        resource_args, opts = _utilities.get_resource_args_opts(
            ExecArgs, pulumi.ResourceOptions, *args, **kwargs
        )
        if resource_args is not None:
            __self__._internal_init(resource_name, opts, **resource_args.__dict__)
        else:
            __self__._internal_init(resource_name, *args, **kwargs)

    def _internal_init(
        __self__,
        resource_name: str,
        opts: Optional[pulumi.ResourceOptions] = None,
        config: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ResourceConfigArgs",
                    "_root_inputs.ResourceConfigArgsDict",
                ]
            ]
        ] = None,
        connection: Optional[
            pulumi.Input[
                Union["_root_inputs.ConnectionArgs", "_root_inputs.ConnectionArgsDict"]
            ]
        ] = None,
        create: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ExecCommandArgs", "_root_inputs.ExecCommandArgsDict"
                ]
            ]
        ] = None,
        delete: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ExecCommandArgs", "_root_inputs.ExecCommandArgsDict"
                ]
            ]
        ] = None,
        delete_before_replace: Optional[pulumi.Input[_builtins.bool]] = None,
        dir: Optional[pulumi.Input[_builtins.str]] = None,
        environment: Optional[
            pulumi.Input[Mapping[str, pulumi.Input[_builtins.str]]]
        ] = None,
        expand_argument_vars: Optional[pulumi.Input[_builtins.bool]] = None,
        logging: Optional[pulumi.Input[_builtins.str]] = None,
        triggers: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.TriggersInputArgs",
                    "_root_inputs.TriggersInputArgsDict",
                ]
            ]
        ] = None,
        update: Optional[
            pulumi.Input[
                Union[
                    "_root_inputs.ExecCommandArgs", "_root_inputs.ExecCommandArgsDict"
                ]
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
            __props__ = ExecArgs.__new__(ExecArgs)

            __props__.__dict__["config"] = config
            __props__.__dict__["connection"] = connection
            if create is None and not opts.urn:
                raise TypeError("Missing required property 'create'")
            __props__.__dict__["create"] = create
            __props__.__dict__["delete"] = delete
            __props__.__dict__["delete_before_replace"] = delete_before_replace
            __props__.__dict__["dir"] = dir
            __props__.__dict__["environment"] = environment
            __props__.__dict__["expand_argument_vars"] = expand_argument_vars
            __props__.__dict__["logging"] = logging
            __props__.__dict__["triggers"] = triggers
            __props__.__dict__["update"] = update
            __props__.__dict__["stderr"] = None
            __props__.__dict__["stdout"] = None
        super(Exec, __self__).__init__(
            "mid:resource:Exec", resource_name, __props__, opts
        )

    @staticmethod
    def get(
        resource_name: str,
        id: pulumi.Input[str],
        opts: Optional[pulumi.ResourceOptions] = None,
    ) -> "Exec":
        """
        Get an existing Exec resource's state with the given name, id, and optional extra
        properties used to qualify the lookup.

        :param str resource_name: The unique name of the resulting resource.
        :param pulumi.Input[str] id: The unique provider ID of the resource to lookup.
        :param pulumi.ResourceOptions opts: Options for the resource.
        """
        opts = pulumi.ResourceOptions.merge(opts, pulumi.ResourceOptions(id=id))

        __props__ = ExecArgs.__new__(ExecArgs)

        __props__.__dict__["config"] = None
        __props__.__dict__["connection"] = None
        __props__.__dict__["create"] = None
        __props__.__dict__["delete"] = None
        __props__.__dict__["delete_before_replace"] = None
        __props__.__dict__["dir"] = None
        __props__.__dict__["environment"] = None
        __props__.__dict__["expand_argument_vars"] = None
        __props__.__dict__["logging"] = None
        __props__.__dict__["stderr"] = None
        __props__.__dict__["stdout"] = None
        __props__.__dict__["triggers"] = None
        __props__.__dict__["update"] = None
        return Exec(resource_name, opts=opts, __props__=__props__)

    @_builtins.property
    @pulumi.getter
    def config(self) -> pulumi.Output[Optional["_root_outputs.ResourceConfig"]]:
        return pulumi.get(self, "config")

    @_builtins.property
    @pulumi.getter
    def connection(self) -> pulumi.Output[Optional["_root_outputs.Connection"]]:
        return pulumi.get(self, "connection")

    @_builtins.property
    @pulumi.getter
    def create(self) -> pulumi.Output["_root_outputs.ExecCommand"]:
        return pulumi.get(self, "create")

    @_builtins.property
    @pulumi.getter
    def delete(self) -> pulumi.Output[Optional["_root_outputs.ExecCommand"]]:
        return pulumi.get(self, "delete")

    @_builtins.property
    @pulumi.getter(name="deleteBeforeReplace")
    def delete_before_replace(self) -> pulumi.Output[Optional[_builtins.bool]]:
        return pulumi.get(self, "delete_before_replace")

    @_builtins.property
    @pulumi.getter
    def dir(self) -> pulumi.Output[Optional[_builtins.str]]:
        return pulumi.get(self, "dir")

    @_builtins.property
    @pulumi.getter
    def environment(self) -> pulumi.Output[Optional[Mapping[str, _builtins.str]]]:
        return pulumi.get(self, "environment")

    @_builtins.property
    @pulumi.getter(name="expandArgumentVars")
    def expand_argument_vars(self) -> pulumi.Output[Optional[_builtins.bool]]:
        return pulumi.get(self, "expand_argument_vars")

    @_builtins.property
    @pulumi.getter
    def logging(self) -> pulumi.Output[Optional[_builtins.str]]:
        return pulumi.get(self, "logging")

    @_builtins.property
    @pulumi.getter
    def stderr(self) -> pulumi.Output[_builtins.str]:
        return pulumi.get(self, "stderr")

    @_builtins.property
    @pulumi.getter
    def stdout(self) -> pulumi.Output[_builtins.str]:
        return pulumi.get(self, "stdout")

    @_builtins.property
    @pulumi.getter
    def triggers(self) -> pulumi.Output["_root_outputs.TriggersOutput"]:
        return pulumi.get(self, "triggers")

    @_builtins.property
    @pulumi.getter
    def update(self) -> pulumi.Output[Optional["_root_outputs.ExecCommand"]]:
        return pulumi.get(self, "update")
