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

__all__ = [
    "AnsibleExecuteResult",
    "AwaitableAnsibleExecuteResult",
    "ansible_execute",
    "ansible_execute_output",
]


@pulumi.output_type
class AnsibleExecuteResult:
    def __init__(
        __self__,
        args=None,
        check=None,
        config=None,
        connection=None,
        debug_keep_temp_files=None,
        debug_temp_dir=None,
        environment=None,
        exit_code=None,
        name=None,
        result=None,
        stderr=None,
        stdout=None,
    ):
        if args and not isinstance(args, dict):
            raise TypeError("Expected argument 'args' to be a dict")
        pulumi.set(__self__, "args", args)
        if check and not isinstance(check, bool):
            raise TypeError("Expected argument 'check' to be a bool")
        pulumi.set(__self__, "check", check)
        if config and not isinstance(config, dict):
            raise TypeError("Expected argument 'config' to be a dict")
        pulumi.set(__self__, "config", config)
        if connection and not isinstance(connection, dict):
            raise TypeError("Expected argument 'connection' to be a dict")
        pulumi.set(__self__, "connection", connection)
        if debug_keep_temp_files and not isinstance(debug_keep_temp_files, bool):
            raise TypeError("Expected argument 'debug_keep_temp_files' to be a bool")
        pulumi.set(__self__, "debug_keep_temp_files", debug_keep_temp_files)
        if debug_temp_dir and not isinstance(debug_temp_dir, str):
            raise TypeError("Expected argument 'debug_temp_dir' to be a str")
        pulumi.set(__self__, "debug_temp_dir", debug_temp_dir)
        if environment and not isinstance(environment, dict):
            raise TypeError("Expected argument 'environment' to be a dict")
        pulumi.set(__self__, "environment", environment)
        if exit_code and not isinstance(exit_code, int):
            raise TypeError("Expected argument 'exit_code' to be a int")
        pulumi.set(__self__, "exit_code", exit_code)
        if name and not isinstance(name, str):
            raise TypeError("Expected argument 'name' to be a str")
        pulumi.set(__self__, "name", name)
        if result and not isinstance(result, dict):
            raise TypeError("Expected argument 'result' to be a dict")
        pulumi.set(__self__, "result", result)
        if stderr and not isinstance(stderr, str):
            raise TypeError("Expected argument 'stderr' to be a str")
        pulumi.set(__self__, "stderr", stderr)
        if stdout and not isinstance(stdout, str):
            raise TypeError("Expected argument 'stdout' to be a str")
        pulumi.set(__self__, "stdout", stdout)

    @_builtins.property
    @pulumi.getter
    def args(self) -> Mapping[str, Any]:
        return pulumi.get(self, "args")

    @_builtins.property
    @pulumi.getter
    def check(self) -> Optional[_builtins.bool]:
        return pulumi.get(self, "check")

    @_builtins.property
    @pulumi.getter
    def config(self) -> Optional["_root_outputs.ResourceConfig"]:
        return pulumi.get(self, "config")

    @_builtins.property
    @pulumi.getter
    def connection(self) -> Optional["_root_outputs.Connection"]:
        return pulumi.get(self, "connection")

    @_builtins.property
    @pulumi.getter(name="debugKeepTempFiles")
    def debug_keep_temp_files(self) -> Optional[_builtins.bool]:
        return pulumi.get(self, "debug_keep_temp_files")

    @_builtins.property
    @pulumi.getter(name="debugTempDir")
    def debug_temp_dir(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "debug_temp_dir")

    @_builtins.property
    @pulumi.getter
    def environment(self) -> Optional[Mapping[str, _builtins.str]]:
        return pulumi.get(self, "environment")

    @_builtins.property
    @pulumi.getter(name="exitCode")
    def exit_code(self) -> _builtins.int:
        return pulumi.get(self, "exit_code")

    @_builtins.property
    @pulumi.getter
    def name(self) -> _builtins.str:
        return pulumi.get(self, "name")

    @_builtins.property
    @pulumi.getter
    def result(self) -> Mapping[str, Any]:
        return pulumi.get(self, "result")

    @_builtins.property
    @pulumi.getter
    def stderr(self) -> _builtins.str:
        return pulumi.get(self, "stderr")

    @_builtins.property
    @pulumi.getter
    def stdout(self) -> _builtins.str:
        return pulumi.get(self, "stdout")


class AwaitableAnsibleExecuteResult(AnsibleExecuteResult):
    # pylint: disable=using-constant-test
    def __await__(self):
        if False:
            yield self
        return AnsibleExecuteResult(
            args=self.args,
            check=self.check,
            config=self.config,
            connection=self.connection,
            debug_keep_temp_files=self.debug_keep_temp_files,
            debug_temp_dir=self.debug_temp_dir,
            environment=self.environment,
            exit_code=self.exit_code,
            name=self.name,
            result=self.result,
            stderr=self.stderr,
            stdout=self.stdout,
        )


def ansible_execute(
    args: Optional[Mapping[str, Any]] = None,
    check: Optional[_builtins.bool] = None,
    config: Optional[
        Union["_root_inputs.ResourceConfig", "_root_inputs.ResourceConfigDict"]
    ] = None,
    connection: Optional[
        Union["_root_inputs.Connection", "_root_inputs.ConnectionDict"]
    ] = None,
    debug_keep_temp_files: Optional[_builtins.bool] = None,
    environment: Optional[Mapping[str, _builtins.str]] = None,
    name: Optional[_builtins.str] = None,
    opts: Optional[pulumi.InvokeOptions] = None,
) -> AwaitableAnsibleExecuteResult:
    """
    Use this data source to access information about an existing resource.
    """
    __args__ = dict()
    __args__["args"] = args
    __args__["check"] = check
    __args__["config"] = config
    __args__["connection"] = connection
    __args__["debugKeepTempFiles"] = debug_keep_temp_files
    __args__["environment"] = environment
    __args__["name"] = name
    opts = pulumi.InvokeOptions.merge(_utilities.get_invoke_opts_defaults(), opts)
    __ret__ = pulumi.runtime.invoke(
        "mid:agent:ansibleExecute", __args__, opts=opts, typ=AnsibleExecuteResult
    ).value

    return AwaitableAnsibleExecuteResult(
        args=pulumi.get(__ret__, "args"),
        check=pulumi.get(__ret__, "check"),
        config=pulumi.get(__ret__, "config"),
        connection=pulumi.get(__ret__, "connection"),
        debug_keep_temp_files=pulumi.get(__ret__, "debug_keep_temp_files"),
        debug_temp_dir=pulumi.get(__ret__, "debug_temp_dir"),
        environment=pulumi.get(__ret__, "environment"),
        exit_code=pulumi.get(__ret__, "exit_code"),
        name=pulumi.get(__ret__, "name"),
        result=pulumi.get(__ret__, "result"),
        stderr=pulumi.get(__ret__, "stderr"),
        stdout=pulumi.get(__ret__, "stdout"),
    )


def ansible_execute_output(
    args: Optional[pulumi.Input[Mapping[str, Any]]] = None,
    check: Optional[pulumi.Input[Optional[_builtins.bool]]] = None,
    config: Optional[
        pulumi.Input[
            Optional[
                Union["_root_inputs.ResourceConfig", "_root_inputs.ResourceConfigDict"]
            ]
        ]
    ] = None,
    connection: Optional[
        pulumi.Input[
            Optional[Union["_root_inputs.Connection", "_root_inputs.ConnectionDict"]]
        ]
    ] = None,
    debug_keep_temp_files: Optional[pulumi.Input[Optional[_builtins.bool]]] = None,
    environment: Optional[pulumi.Input[Optional[Mapping[str, _builtins.str]]]] = None,
    name: Optional[pulumi.Input[_builtins.str]] = None,
    opts: Optional[Union[pulumi.InvokeOptions, pulumi.InvokeOutputOptions]] = None,
) -> pulumi.Output[AnsibleExecuteResult]:
    """
    Use this data source to access information about an existing resource.
    """
    __args__ = dict()
    __args__["args"] = args
    __args__["check"] = check
    __args__["config"] = config
    __args__["connection"] = connection
    __args__["debugKeepTempFiles"] = debug_keep_temp_files
    __args__["environment"] = environment
    __args__["name"] = name
    opts = pulumi.InvokeOutputOptions.merge(_utilities.get_invoke_opts_defaults(), opts)
    __ret__ = pulumi.runtime.invoke_output(
        "mid:agent:ansibleExecute", __args__, opts=opts, typ=AnsibleExecuteResult
    )
    return __ret__.apply(
        lambda __response__: AnsibleExecuteResult(
            args=pulumi.get(__response__, "args"),
            check=pulumi.get(__response__, "check"),
            config=pulumi.get(__response__, "config"),
            connection=pulumi.get(__response__, "connection"),
            debug_keep_temp_files=pulumi.get(__response__, "debug_keep_temp_files"),
            debug_temp_dir=pulumi.get(__response__, "debug_temp_dir"),
            environment=pulumi.get(__response__, "environment"),
            exit_code=pulumi.get(__response__, "exit_code"),
            name=pulumi.get(__response__, "name"),
            result=pulumi.get(__response__, "result"),
            stderr=pulumi.get(__response__, "stderr"),
            stdout=pulumi.get(__response__, "stdout"),
        )
    )
