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
    "FileStatResult",
    "AwaitableFileStatResult",
    "file_stat",
    "file_stat_output",
]


@pulumi.output_type
class FileStatResult:
    def __init__(
        __self__,
        access_time=None,
        base_name=None,
        calculate_checksum=None,
        config=None,
        connection=None,
        create_time=None,
        dev=None,
        exists=None,
        file_mode=None,
        follow_symlinks=None,
        gid=None,
        group_name=None,
        inode=None,
        modified_time=None,
        nlink=None,
        path=None,
        sha256_checksum=None,
        size=None,
        uid=None,
        user_name=None,
    ):
        if access_time and not isinstance(access_time, str):
            raise TypeError("Expected argument 'access_time' to be a str")
        pulumi.set(__self__, "access_time", access_time)
        if base_name and not isinstance(base_name, str):
            raise TypeError("Expected argument 'base_name' to be a str")
        pulumi.set(__self__, "base_name", base_name)
        if calculate_checksum and not isinstance(calculate_checksum, bool):
            raise TypeError("Expected argument 'calculate_checksum' to be a bool")
        pulumi.set(__self__, "calculate_checksum", calculate_checksum)
        if config and not isinstance(config, dict):
            raise TypeError("Expected argument 'config' to be a dict")
        pulumi.set(__self__, "config", config)
        if connection and not isinstance(connection, dict):
            raise TypeError("Expected argument 'connection' to be a dict")
        pulumi.set(__self__, "connection", connection)
        if create_time and not isinstance(create_time, str):
            raise TypeError("Expected argument 'create_time' to be a str")
        pulumi.set(__self__, "create_time", create_time)
        if dev and not isinstance(dev, int):
            raise TypeError("Expected argument 'dev' to be a int")
        pulumi.set(__self__, "dev", dev)
        if exists and not isinstance(exists, bool):
            raise TypeError("Expected argument 'exists' to be a bool")
        pulumi.set(__self__, "exists", exists)
        if file_mode and not isinstance(file_mode, dict):
            raise TypeError("Expected argument 'file_mode' to be a dict")
        pulumi.set(__self__, "file_mode", file_mode)
        if follow_symlinks and not isinstance(follow_symlinks, bool):
            raise TypeError("Expected argument 'follow_symlinks' to be a bool")
        pulumi.set(__self__, "follow_symlinks", follow_symlinks)
        if gid and not isinstance(gid, int):
            raise TypeError("Expected argument 'gid' to be a int")
        pulumi.set(__self__, "gid", gid)
        if group_name and not isinstance(group_name, str):
            raise TypeError("Expected argument 'group_name' to be a str")
        pulumi.set(__self__, "group_name", group_name)
        if inode and not isinstance(inode, int):
            raise TypeError("Expected argument 'inode' to be a int")
        pulumi.set(__self__, "inode", inode)
        if modified_time and not isinstance(modified_time, str):
            raise TypeError("Expected argument 'modified_time' to be a str")
        pulumi.set(__self__, "modified_time", modified_time)
        if nlink and not isinstance(nlink, int):
            raise TypeError("Expected argument 'nlink' to be a int")
        pulumi.set(__self__, "nlink", nlink)
        if path and not isinstance(path, str):
            raise TypeError("Expected argument 'path' to be a str")
        pulumi.set(__self__, "path", path)
        if sha256_checksum and not isinstance(sha256_checksum, str):
            raise TypeError("Expected argument 'sha256_checksum' to be a str")
        pulumi.set(__self__, "sha256_checksum", sha256_checksum)
        if size and not isinstance(size, int):
            raise TypeError("Expected argument 'size' to be a int")
        pulumi.set(__self__, "size", size)
        if uid and not isinstance(uid, int):
            raise TypeError("Expected argument 'uid' to be a int")
        pulumi.set(__self__, "uid", uid)
        if user_name and not isinstance(user_name, str):
            raise TypeError("Expected argument 'user_name' to be a str")
        pulumi.set(__self__, "user_name", user_name)

    @_builtins.property
    @pulumi.getter(name="accessTime")
    def access_time(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "access_time")

    @_builtins.property
    @pulumi.getter(name="baseName")
    def base_name(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "base_name")

    @_builtins.property
    @pulumi.getter(name="calculateChecksum")
    def calculate_checksum(self) -> Optional[_builtins.bool]:
        return pulumi.get(self, "calculate_checksum")

    @_builtins.property
    @pulumi.getter
    def config(self) -> Optional["_root_outputs.ResourceConfig"]:
        return pulumi.get(self, "config")

    @_builtins.property
    @pulumi.getter
    def connection(self) -> Optional["_root_outputs.Connection"]:
        return pulumi.get(self, "connection")

    @_builtins.property
    @pulumi.getter(name="createTime")
    def create_time(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "create_time")

    @_builtins.property
    @pulumi.getter
    def dev(self) -> Optional[_builtins.int]:
        return pulumi.get(self, "dev")

    @_builtins.property
    @pulumi.getter
    def exists(self) -> _builtins.bool:
        return pulumi.get(self, "exists")

    @_builtins.property
    @pulumi.getter(name="fileMode")
    def file_mode(self) -> Optional["_root_outputs.FileStatFileMode"]:
        return pulumi.get(self, "file_mode")

    @_builtins.property
    @pulumi.getter(name="followSymlinks")
    def follow_symlinks(self) -> Optional[_builtins.bool]:
        return pulumi.get(self, "follow_symlinks")

    @_builtins.property
    @pulumi.getter
    def gid(self) -> Optional[_builtins.int]:
        return pulumi.get(self, "gid")

    @_builtins.property
    @pulumi.getter(name="groupName")
    def group_name(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "group_name")

    @_builtins.property
    @pulumi.getter
    def inode(self) -> Optional[_builtins.int]:
        return pulumi.get(self, "inode")

    @_builtins.property
    @pulumi.getter(name="modifiedTime")
    def modified_time(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "modified_time")

    @_builtins.property
    @pulumi.getter
    def nlink(self) -> Optional[_builtins.int]:
        return pulumi.get(self, "nlink")

    @_builtins.property
    @pulumi.getter
    def path(self) -> _builtins.str:
        return pulumi.get(self, "path")

    @_builtins.property
    @pulumi.getter(name="sha256Checksum")
    def sha256_checksum(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "sha256_checksum")

    @_builtins.property
    @pulumi.getter
    def size(self) -> Optional[_builtins.int]:
        return pulumi.get(self, "size")

    @_builtins.property
    @pulumi.getter
    def uid(self) -> Optional[_builtins.int]:
        return pulumi.get(self, "uid")

    @_builtins.property
    @pulumi.getter(name="userName")
    def user_name(self) -> Optional[_builtins.str]:
        return pulumi.get(self, "user_name")


class AwaitableFileStatResult(FileStatResult):
    # pylint: disable=using-constant-test
    def __await__(self):
        if False:
            yield self
        return FileStatResult(
            access_time=self.access_time,
            base_name=self.base_name,
            calculate_checksum=self.calculate_checksum,
            config=self.config,
            connection=self.connection,
            create_time=self.create_time,
            dev=self.dev,
            exists=self.exists,
            file_mode=self.file_mode,
            follow_symlinks=self.follow_symlinks,
            gid=self.gid,
            group_name=self.group_name,
            inode=self.inode,
            modified_time=self.modified_time,
            nlink=self.nlink,
            path=self.path,
            sha256_checksum=self.sha256_checksum,
            size=self.size,
            uid=self.uid,
            user_name=self.user_name,
        )


def file_stat(
    calculate_checksum: Optional[_builtins.bool] = None,
    config: Optional[
        Union["_root_inputs.ResourceConfig", "_root_inputs.ResourceConfigDict"]
    ] = None,
    connection: Optional[
        Union["_root_inputs.Connection", "_root_inputs.ConnectionDict"]
    ] = None,
    follow_symlinks: Optional[_builtins.bool] = None,
    path: Optional[_builtins.str] = None,
    opts: Optional[pulumi.InvokeOptions] = None,
) -> AwaitableFileStatResult:
    """
    Use this data source to access information about an existing resource.
    """
    __args__ = dict()
    __args__["calculateChecksum"] = calculate_checksum
    __args__["config"] = config
    __args__["connection"] = connection
    __args__["followSymlinks"] = follow_symlinks
    __args__["path"] = path
    opts = pulumi.InvokeOptions.merge(_utilities.get_invoke_opts_defaults(), opts)
    __ret__ = pulumi.runtime.invoke(
        "mid:agent:fileStat", __args__, opts=opts, typ=FileStatResult
    ).value

    return AwaitableFileStatResult(
        access_time=pulumi.get(__ret__, "access_time"),
        base_name=pulumi.get(__ret__, "base_name"),
        calculate_checksum=pulumi.get(__ret__, "calculate_checksum"),
        config=pulumi.get(__ret__, "config"),
        connection=pulumi.get(__ret__, "connection"),
        create_time=pulumi.get(__ret__, "create_time"),
        dev=pulumi.get(__ret__, "dev"),
        exists=pulumi.get(__ret__, "exists"),
        file_mode=pulumi.get(__ret__, "file_mode"),
        follow_symlinks=pulumi.get(__ret__, "follow_symlinks"),
        gid=pulumi.get(__ret__, "gid"),
        group_name=pulumi.get(__ret__, "group_name"),
        inode=pulumi.get(__ret__, "inode"),
        modified_time=pulumi.get(__ret__, "modified_time"),
        nlink=pulumi.get(__ret__, "nlink"),
        path=pulumi.get(__ret__, "path"),
        sha256_checksum=pulumi.get(__ret__, "sha256_checksum"),
        size=pulumi.get(__ret__, "size"),
        uid=pulumi.get(__ret__, "uid"),
        user_name=pulumi.get(__ret__, "user_name"),
    )


def file_stat_output(
    calculate_checksum: Optional[pulumi.Input[Optional[_builtins.bool]]] = None,
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
    follow_symlinks: Optional[pulumi.Input[Optional[_builtins.bool]]] = None,
    path: Optional[pulumi.Input[_builtins.str]] = None,
    opts: Optional[Union[pulumi.InvokeOptions, pulumi.InvokeOutputOptions]] = None,
) -> pulumi.Output[FileStatResult]:
    """
    Use this data source to access information about an existing resource.
    """
    __args__ = dict()
    __args__["calculateChecksum"] = calculate_checksum
    __args__["config"] = config
    __args__["connection"] = connection
    __args__["followSymlinks"] = follow_symlinks
    __args__["path"] = path
    opts = pulumi.InvokeOutputOptions.merge(_utilities.get_invoke_opts_defaults(), opts)
    __ret__ = pulumi.runtime.invoke_output(
        "mid:agent:fileStat", __args__, opts=opts, typ=FileStatResult
    )
    return __ret__.apply(
        lambda __response__: FileStatResult(
            access_time=pulumi.get(__response__, "access_time"),
            base_name=pulumi.get(__response__, "base_name"),
            calculate_checksum=pulumi.get(__response__, "calculate_checksum"),
            config=pulumi.get(__response__, "config"),
            connection=pulumi.get(__response__, "connection"),
            create_time=pulumi.get(__response__, "create_time"),
            dev=pulumi.get(__response__, "dev"),
            exists=pulumi.get(__response__, "exists"),
            file_mode=pulumi.get(__response__, "file_mode"),
            follow_symlinks=pulumi.get(__response__, "follow_symlinks"),
            gid=pulumi.get(__response__, "gid"),
            group_name=pulumi.get(__response__, "group_name"),
            inode=pulumi.get(__response__, "inode"),
            modified_time=pulumi.get(__response__, "modified_time"),
            nlink=pulumi.get(__response__, "nlink"),
            path=pulumi.get(__response__, "path"),
            sha256_checksum=pulumi.get(__response__, "sha256_checksum"),
            size=pulumi.get(__response__, "size"),
            uid=pulumi.get(__response__, "uid"),
            user_name=pulumi.get(__response__, "user_name"),
        )
    )
