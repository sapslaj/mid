# (c) 2013-2014, Michael DeHaan <michael.dehaan@gmail.com>
# (c) 2015 Toshio Kuratomi <tkuratomi@ansible.com>
#
# This file is part of Ansible
#
# Ansible is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Ansible is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Ansible.  If not, see <http://www.gnu.org/licenses/>.

from __future__ import annotations

import ast
import os
import time
from typing import Sequence
import zipfile
import pkgutil

from ast import AST, Import, ImportFrom

from ansible.release import __version__, __author__
from ansible.errors import AnsibleError
from ansible.module_utils.common.text.converters import to_bytes, to_text

from ansible.utils.display import Display
from collections import namedtuple

import importlib.machinery

display = Display()

ModuleUtilsProcessEntry = namedtuple(
    "ModuleUtilsProcessEntry",
    ["name_parts", "is_ambiguous", "has_redirected_child", "is_optional"],
)

REPLACER = b"#<<INCLUDE_ANSIBLE_MODULE_COMMON>>"
REPLACER_VERSION = b'"<<ANSIBLE_VERSION>>"'
REPLACER_COMPLEX = b'"<<INCLUDE_ANSIBLE_MODULE_COMPLEX_ARGS>>"'
REPLACER_WINDOWS = b"# POWERSHELL_COMMON"
REPLACER_JSONARGS = b"<<INCLUDE_ANSIBLE_MODULE_JSON_ARGS>>"
REPLACER_SELINUX = b"<<SELINUX_SPECIAL_FILESYSTEMS>>"

# We could end up writing out parameters with unicode characters so we need to
# specify an encoding for the python source file
ENCODING_STRING = "# -*- coding: utf-8 -*-"
b_ENCODING_STRING = b"# -*- coding: utf-8 -*-"

# module_common is relative to module_utils, so fix the path
_MODULE_UTILS_PATH = os.path.join(os.path.dirname(__file__), "..", "module_utils")


class ModuleDepFinder(ast.NodeVisitor):
    submodules: set[tuple[str, ...]]
    optional_imports: set[tuple[str, ...]]
    module_fqn: str
    is_pkg_init: bool

    def __init__(
        self,
        module_fqn: str,
        tree,
        is_pkg_init: bool = False,
        *args,
        **kwargs,
    ):
        """
        Walk the ast tree for the python module.
        :arg module_fqn: The fully qualified name to reach this module in dotted notation.
            example: ansible.module_utils.basic
        :arg is_pkg_init: Inform the finder it's looking at a package init (eg __init__.py) to allow
            relative import expansion to use the proper package level without having imported it locally first.

        Save submodule[.submoduleN][.identifier] into self.submodules
        when they are from ansible.module_utils or ansible_collections packages

        self.submodules will end up with tuples like:
          - ('ansible', 'module_utils', 'basic',)
          - ('ansible', 'module_utils', 'urls', 'fetch_url')
          - ('ansible', 'module_utils', 'database', 'postgres')
          - ('ansible', 'module_utils', 'database', 'postgres', 'quote')
          - ('ansible', 'module_utils', 'database', 'postgres', 'quote')
          - ('ansible_collections', 'my_ns', 'my_col', 'plugins', 'module_utils', 'foo')

        It's up to calling code to determine whether the final element of the
        tuple are module names or something else (function, class, or variable names)
        .. seealso:: :python3:class:`ast.NodeVisitor`
        """
        super(ModuleDepFinder, self).__init__(*args, **kwargs)
        self._tree = tree  # squirrel this away so we can compare node parents to it
        self.submodules = set()
        self.optional_imports = set()
        self.module_fqn = module_fqn
        self.is_pkg_init = is_pkg_init

        self._visit_map = {
            Import: self.visit_Import,
            ImportFrom: self.visit_ImportFrom,
        }

        self.visit(tree)

    def generic_visit(self, node: ast.AST):
        """Overridden ``generic_visit`` that makes some assumptions about our
        use case, and improves performance by calling visitors directly instead
        of calling ``visit`` to offload calling visitors.
        """
        generic_visit = self.generic_visit
        visit_map = self._visit_map
        for field, value in ast.iter_fields(node):
            if isinstance(value, list):
                for item in value:
                    if isinstance(item, (Import, ImportFrom)):
                        item.parent = node
                        visit_map[item.__class__](item)
                    elif isinstance(item, AST):
                        generic_visit(item)

    visit = generic_visit

    def visit_Import(self, node: ast.Import):
        """
        Handle import ansible.module_utils.MODLIB[.MODLIBn] [as asname]

        We save these as interesting submodules when the imported library is in ansible.module_utils
        or ansible.collections
        """
        for alias in node.names:
            if alias.name.startswith("ansible.module_utils.") or alias.name.startswith(
                "ansible_collections."
            ):
                py_mod = tuple(alias.name.split("."))
                self.submodules.add(py_mod)
                # if the import's parent is the root document, it's a required import, otherwise it's optional
                if node.parent != self._tree:
                    self.optional_imports.add(py_mod)
        self.generic_visit(node)

    def visit_ImportFrom(self, node: ast.ImportFrom):
        """
        Handle from ansible.module_utils.MODLIB import [.MODLIBn] [as asname]

        Also has to handle relative imports

        We save these as interesting submodules when the imported library is in ansible.module_utils
        or ansible.collections
        """

        node_module = ""
        # FIXME: These should all get skipped:
        # from ansible.executor import module_common
        # from ...executor import module_common
        # from ... import executor (Currently it gives a non-helpful error)
        if node.level > 0:
            # if we're in a package init, we have to add one to the node level (and make it none if 0 to preserve the right slicing behavior)
            level_slice_offset = (
                -node.level + 1 or None if self.is_pkg_init else -node.level
            )
            if self.module_fqn:
                parts = tuple(self.module_fqn.split("."))
                if node.module:
                    # relative import: from .module import x
                    node_module = ".".join(parts[:level_slice_offset] + (node.module,))
                else:
                    # relative import: from . import x
                    node_module = ".".join(parts[:level_slice_offset])
            else:
                # fall back to an absolute import
                node_module = node.module or ""
        else:
            # absolute import: from module import x
            node_module = node.module or ""

        # Specialcase: six is a special case because of its
        # import logic
        py_mod = None
        if node.names[0].name == "_six":
            self.submodules.add(("_six",))
        elif node_module.startswith("ansible.module_utils"):
            # from ansible.module_utils.MODULE1[.MODULEn] import IDENTIFIER [as asname]
            # from ansible.module_utils.MODULE1[.MODULEn] import MODULEn+1 [as asname]
            # from ansible.module_utils.MODULE1[.MODULEn] import MODULEn+1 [,IDENTIFIER] [as asname]
            # from ansible.module_utils import MODULE1 [,MODULEn] [as asname]
            py_mod = tuple(node_module.split("."))

        elif node_module.startswith("ansible_collections."):
            if (
                node_module.endswith("plugins.module_utils")
                or ".plugins.module_utils." in node_module
            ):
                # from ansible_collections.ns.coll.plugins.module_utils import MODULE [as aname] [,MODULE2] [as aname]
                # from ansible_collections.ns.coll.plugins.module_utils.MODULE import IDENTIFIER [as aname]
                # FIXME: Unhandled cornercase (needs to be ignored):
                # from ansible_collections.ns.coll.plugins.[!module_utils].[FOO].plugins.module_utils import IDENTIFIER
                py_mod = tuple(node_module.split("."))
            else:
                # Not from module_utils so ignore.  for instance:
                # from ansible_collections.ns.coll.plugins.lookup import IDENTIFIER
                pass

        if py_mod:
            for alias in node.names:
                self.submodules.add(py_mod + (alias.name,))
                # if the import's parent is the root document, it's a required import, otherwise it's optional
                if node.parent != self._tree:
                    self.optional_imports.add(py_mod + (alias.name,))

        self.generic_visit(node)


def _slurp(path: str) -> bytes:
    if not os.path.exists(path):
        raise AnsibleError(
            "imported module support code does not exist at %s" % os.path.abspath(path)
        )
    with open(path, "rb") as fd:
        data = fd.read()
    return data


class ModuleUtilLocatorBase:
    _is_ambiguous: bool
    _child_is_redirected: bool
    _is_optional: bool
    found: bool
    redirected: bool
    fq_name_parts: tuple[str, ...]
    source_code: str | bytes
    output_path: str
    is_package: bool
    _collection_name: str | None
    candidate_names: Sequence[tuple[str, ...]]

    def __init__(
        self,
        fq_name_parts: tuple[str, ...],
        is_ambiguous: bool = False,
        child_is_redirected: bool = False,
        is_optional: bool = False,
    ):
        self._is_ambiguous = is_ambiguous
        # a child package redirection could cause intermediate package levels to be missing, eg
        # from ansible.module_utils.x.y.z import foo; if x.y.z.foo is redirected, we may not have packages on disk for
        # the intermediate packages x.y.z, so we'll need to supply empty packages for those
        self._child_is_redirected = child_is_redirected
        self._is_optional = is_optional
        self.found = False
        self.redirected = False
        self.fq_name_parts = fq_name_parts
        self.source_code = ""
        self.output_path = ""
        self.is_package = False
        self._collection_name = None
        # for ambiguous imports, we should only test for things more than one level below module_utils
        # this lets us detect erroneous imports and redirections earlier
        if (
            is_ambiguous
            and len(self._get_module_utils_remainder_parts(fq_name_parts)) > 1
        ):
            self.candidate_names = [fq_name_parts, fq_name_parts[:-1]]
        else:
            self.candidate_names = [fq_name_parts]

    @property
    def candidate_names_joined(self) -> list[str]:
        return [".".join(n) for n in self.candidate_names]

    def _get_module_utils_remainder_parts(self, name_parts: Sequence[str]) -> list[str]:
        # subclasses should override to return the name parts after module_utils
        return []

    def _get_module_utils_remainder(self, name_parts: Sequence[str]) -> str:
        # return the remainder parts as a package string
        return ".".join(self._get_module_utils_remainder_parts(name_parts))

    def _find_module(self, name_parts: Sequence[str]) -> bool:
        return False

    def _locate(self, redirect_first: bool = True) -> None:
        candidate_name_parts: tuple[str, ...] = tuple()
        for candidate_name_parts in self.candidate_names:
            if self._find_module(candidate_name_parts):
                break

        else:  # didn't find what we were looking for- last chance for packages whose parents were redirected
            if self._child_is_redirected:  # make fake packages
                self.is_package = True
                self.source_code = ""
            else:  # nope, just bail
                return

        if self.is_package:
            path_parts = candidate_name_parts + ("__init__",)
        else:
            path_parts = candidate_name_parts
        self.found = True
        self.output_path = os.path.join(*path_parts) + ".py"
        self.fq_name_parts = candidate_name_parts

    def _generate_redirect_shim_source(
        self,
        fq_source_module: str,
        fq_target_module: str,
    ) -> str:
        return """
import sys
import {1} as mod

sys.modules['{0}'] = mod
""".format(fq_source_module, fq_target_module)

        # FIXME: add __repr__ impl


class LegacyModuleUtilLocator(ModuleUtilLocatorBase):
    def __init__(
        self,
        fq_name_parts: tuple[str, ...],
        is_ambiguous: bool = False,
        child_is_redirected: bool = False,
        mu_paths: Sequence[str] | None = None,
    ):
        super(LegacyModuleUtilLocator, self).__init__(
            fq_name_parts, is_ambiguous, child_is_redirected
        )

        if fq_name_parts[0:2] != ("ansible", "module_utils"):
            raise Exception(
                "this class can only locate from ansible.module_utils, got {0}".format(
                    fq_name_parts
                )
            )

        if fq_name_parts[2] == "six":
            # FIXME: handle the ansible.module_utils.six._six case with a redirect or an internal _six attr on six itself?
            # six creates its submodules at runtime; convert all these to just 'ansible.module_utils.six'
            fq_name_parts = ("ansible", "module_utils", "six")
            self.candidate_names = [fq_name_parts]

        self._mu_paths = mu_paths
        self._collection_name = "ansible.builtin"  # legacy module utils always look in ansible.builtin for redirects
        self._locate(
            redirect_first=False
        )  # let local stuff override redirects for legacy

    def _get_module_utils_remainder_parts(self, name_parts):
        return name_parts[2:]  # eg, foo.bar for ansible.module_utils.foo.bar

    def _find_module(self, name_parts):
        rel_name_parts = self._get_module_utils_remainder_parts(name_parts)

        paths = None
        # no redirection; try to find the module
        if (
            len(rel_name_parts) == 1
        ):  # direct child of module_utils, just search the top-level dirs we were given
            paths = self._mu_paths
        elif (
            self._mu_paths is not None
        ):  # a nested submodule of module_utils, extend the paths given with the intermediate package names
            paths = [
                os.path.join(p, *rel_name_parts[:-1]) for p in self._mu_paths
            ]  # extend the MU paths with the relative bit

        # find_spec needs the full module name
        self._info = info = importlib.machinery.PathFinder.find_spec(
            ".".join(name_parts), paths
        )
        if (
            info is not None
            and info.origin is not None
            and os.path.splitext(info.origin)[1] in importlib.machinery.SOURCE_SUFFIXES
        ):
            self.is_package = info.origin.endswith("/__init__.py")
            path = info.origin
        else:
            return False
        self.source_code = _slurp(path)

        return True


class CollectionModuleUtilLocator(ModuleUtilLocatorBase):
    def __init__(
        self,
        fq_name_parts: tuple[str, ...],
        is_ambiguous: bool = False,
        child_is_redirected: bool = False,
        is_optional: bool = False,
    ):
        super(CollectionModuleUtilLocator, self).__init__(
            fq_name_parts=fq_name_parts,
            is_ambiguous=is_ambiguous,
            child_is_redirected=child_is_redirected,
            is_optional=is_optional,
        )

        if fq_name_parts[0] != "ansible_collections":
            raise Exception(
                "CollectionModuleUtilLocator can only locate from ansible_collections, got {0}".format(
                    fq_name_parts
                )
            )
        elif len(fq_name_parts) >= 6 and fq_name_parts[3:5] != (
            "plugins",
            "module_utils",
        ):
            raise Exception(
                "CollectionModuleUtilLocator can only locate below ansible_collections.(ns).(coll).plugins.module_utils, got {0}".format(
                    fq_name_parts
                )
            )

        self._collection_name = ".".join(fq_name_parts[1:3])

        self._locate()

    def _find_module(self, name_parts: Sequence[str]) -> bool:
        # synthesize empty inits for packages down through module_utils- we don't want to allow those to be shipped over, but the
        # package hierarchy needs to exist
        if len(name_parts) < 6:
            self.source_code = ""
            self.is_package = True
            return True

        # NB: we can't use pkgutil.get_data safely here, since we don't want to import/execute package/module code on
        # the controller while analyzing/assembling the module, so we'll have to manually import the collection's
        # Python package to locate it (import root collection, reassemble resource path beneath, fetch source)

        collection_pkg_name = ".".join(name_parts[0:3])
        resource_base_path = os.path.join(*name_parts[3:])

        src = None
        # look for package_dir first, then module
        try:
            src = pkgutil.get_data(
                collection_pkg_name,
                to_native(os.path.join(resource_base_path, "__init__.py")),
            )
        except ImportError:
            pass

        # TODO: we might want to synthesize fake inits for py3-style packages, for now they're required beneath module_utils

        if src is not None:  # empty string is OK
            self.is_package = True
        else:
            try:
                src = pkgutil.get_data(
                    collection_pkg_name, to_native(resource_base_path + ".py")
                )
            except ImportError:
                pass

        if src is None:  # empty string is OK
            return False

        self.source_code = src
        return True

    def _get_module_utils_remainder_parts(
        self,
        name_parts: Sequence[str],
    ) -> Sequence[str]:
        return name_parts[
            5:
        ]  # eg, foo.bar for ansible_collections.ns.coll.plugins.module_utils.foo.bar


def _make_zinfo(
    filename: str,
    date_time: tuple[int, int, int, int, int, int],
    zf: zipfile.ZipFile | None = None,
) -> zipfile.ZipInfo:
    zinfo = zipfile.ZipInfo(filename=filename, date_time=date_time)
    if zf:
        zinfo.compress_type = zf.compression
    return zinfo


def recursive_finder(
    name: str,
    module_fqn: str,
    module_data: bytes,
    zf: zipfile.ZipFile,
    date_time: tuple[int, int, int, int, int, int] | None = None,
) -> None:
    """
    Using ModuleDepFinder, make sure we have all of the module_utils files that
    the module and its module_utils files needs. (no longer actually recursive)
    :arg name: Name of the python module we're examining
    :arg module_fqn: Fully qualified name of the python module we're scanning
    :arg module_data: string Python code of the module we're scanning
    :arg zf: An open :python:class:`zipfile.ZipFile` object that holds the Ansible module payload
        which we're assembling
    """
    if date_time is None:
        date_time = time.gmtime()[:6]

    # py_module_cache maps python module names to a tuple of the code in the module
    # and the pathname to the module.
    # Here we pre-load it with modules which we create without bothering to
    # read from actual files (In some cases, these need to differ from what ansible
    # ships because they're namespace packages in the module)
    # FIXME: do we actually want ns pkg behavior for these? Seems like they should just be forced to emptyish pkg stubs
    py_module_cache = {
        ("ansible",): (
            b"from pkgutil import extend_path\n"
            b"__path__=extend_path(__path__,__name__)\n"
            b'__version__="'
            + to_bytes(__version__)
            + b'"\n__author__="'
            + to_bytes(__author__)
            + b'"\n',
            "ansible/__init__.py",
        ),
        ("ansible", "module_utils"): (
            b"from pkgutil import extend_path\n"
            b"__path__=extend_path(__path__,__name__)\n",
            "ansible/module_utils/__init__.py",
        ),
    }

    module_utils_paths = [_MODULE_UTILS_PATH]

    # Parse the module code and find the imports of ansible.module_utils
    try:
        tree = compile(module_data, "<unknown>", "exec", ast.PyCF_ONLY_AST)
    except (SyntaxError, IndentationError) as e:
        raise AnsibleError("Unable to import %s due to %s" % (name, e.msg))

    finder = ModuleDepFinder(module_fqn, tree)

    # the format of this set is a tuple of the module name and whether or not the import is ambiguous as a module name
    # or an attribute of a module (eg from x.y import z <-- is z a module or an attribute of x.y?)
    modules_to_process = [
        ModuleUtilsProcessEntry(
            m, True, False, is_optional=m in finder.optional_imports
        )
        for m in finder.submodules
    ]

    # HACK: basic is currently always required since module global init is currently tied up with AnsiballZ arg input
    modules_to_process.append(
        ModuleUtilsProcessEntry(
            ("ansible", "module_utils", "basic"), False, False, is_optional=False
        )
    )

    # we'll be adding new modules inline as we discover them, so just keep going til we've processed them all
    while modules_to_process:
        modules_to_process.sort()  # not strictly necessary, but nice to process things in predictable and repeatable order
        py_module_name, is_ambiguous, child_is_redirected, is_optional = (
            modules_to_process.pop(0)
        )

        if py_module_name in py_module_cache:
            # this is normal; we'll often see the same module imported many times, but we only need to process it once
            continue

        if py_module_name[0:2] == ("ansible", "module_utils"):
            module_info = LegacyModuleUtilLocator(
                py_module_name,
                is_ambiguous=is_ambiguous,
                mu_paths=module_utils_paths,
                child_is_redirected=child_is_redirected,
            )
        elif py_module_name[0] == "ansible_collections":
            module_info = CollectionModuleUtilLocator(
                py_module_name,
                is_ambiguous=is_ambiguous,
                child_is_redirected=child_is_redirected,
                is_optional=is_optional,
            )
        else:
            # FIXME: dot-joined result
            display.warning(
                "ModuleDepFinder improperly found a non-module_utils import %s"
                % [py_module_name]
            )
            continue

        # Could not find the module.  Construct a helpful error message.
        if not module_info.found:
            if is_optional:
                # this was a best-effort optional import that we couldn't find, oh well, move along...
                continue
            # FIXME: use dot-joined candidate names
            msg = "Could not find imported module support code for {0}.  Looked for ({1})".format(
                module_fqn, module_info.candidate_names_joined
            )
            raise AnsibleError(msg)

        # check the cache one more time with the module we actually found, since the name could be different than the input
        # eg, imported name vs module
        if module_info.fq_name_parts in py_module_cache:
            continue

        # compile the source, process all relevant imported modules
        try:
            tree = compile(
                module_info.source_code, "<unknown>", "exec", ast.PyCF_ONLY_AST
            )
        except (SyntaxError, IndentationError) as e:
            raise AnsibleError(
                "Unable to import %s due to %s" % (module_info.fq_name_parts, e.msg)
            )

        finder = ModuleDepFinder(
            ".".join(module_info.fq_name_parts), tree, module_info.is_package
        )
        modules_to_process.extend(
            ModuleUtilsProcessEntry(
                m, True, False, is_optional=m in finder.optional_imports
            )
            for m in finder.submodules
            if m not in py_module_cache
        )

        # we've processed this item, add it to the output list
        py_module_cache[module_info.fq_name_parts] = (
            module_info.source_code,
            module_info.output_path,
        )

        # ensure we process all ancestor package inits
        accumulated_pkg_name = []
        for pkg in module_info.fq_name_parts[:-1]:
            accumulated_pkg_name.append(
                pkg
            )  # we're accumulating this across iterations
            normalized_name = tuple(
                accumulated_pkg_name
            )  # extra machinations to get a hashable type (list is not)
            if normalized_name not in py_module_cache:
                modules_to_process.append(
                    ModuleUtilsProcessEntry(
                        normalized_name,
                        False,
                        module_info.redirected,
                        is_optional=is_optional,
                    )
                )

    for py_module_name in py_module_cache:
        py_module_file_name = py_module_cache[py_module_name][1]

        zf.writestr(
            _make_zinfo(py_module_file_name, date_time, zf=zf),
            py_module_cache[py_module_name][0],
        )
        mu_file = to_text(py_module_file_name, errors="surrogate_or_strict")
        display.vvvvv("Including module_utils file %s" % mu_file)


def get_action_args_with_defaults(action, args, defaults, templar, action_groups=None):
    # Get the list of groups that contain this action
    if action_groups is None:
        msg = (
            "Finding module_defaults for action %s. "
            "The caller has not passed the action_groups, so any "
            "that may include this action will be ignored."
        )
        display.warning(msg=msg)
        group_names = []
    else:
        group_names = action_groups.get(action, [])

    tmp_args = {}
    module_defaults = {}

    # Merge latest defaults into dict, since they are a list of dicts
    if isinstance(defaults, list):
        for default in defaults:
            module_defaults.update(default)

    # module_defaults keys are static, but the values may be templated
    module_defaults = templar.template(module_defaults)
    for default in module_defaults:
        if default.startswith("group/"):
            group_name = default.split("group/")[-1]
            if group_name in group_names:
                tmp_args.update(
                    (module_defaults.get("group/%s" % group_name) or {}).copy()
                )

    # handle specific action defaults
    tmp_args.update(module_defaults.get(action, {}).copy())

    # direct args override all
    tmp_args.update(args)

    return tmp_args
