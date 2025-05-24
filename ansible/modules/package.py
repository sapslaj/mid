# -*- coding: utf-8 -*-

# (c) 2015, Ansible Project
#
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)

from __future__ import annotations


DOCUMENTATION = """
---
module: package
version_added: 2.0
author:
    - Ansible Core Team
short_description: Generic OS package manager
description:
    - This modules manages packages on a target without specifying a package manager module (like M(ansible.builtin.dnf), M(ansible.builtin.apt), ...).
      It is convenient to use in an heterogeneous environment of machines without having to create a specific task for
      each package manager. M(ansible.builtin.package) calls behind the module for the package manager used by the operating system
      discovered by the module M(ansible.builtin.setup).  If M(ansible.builtin.setup) was not yet run, M(ansible.builtin.package) will run it.
    - This module acts as a proxy to the underlying package manager module. While all arguments will be passed to the
      underlying module, not all modules support the same arguments. This documentation only covers the minimum intersection
      of module arguments that all packaging modules support.
    - For Windows targets, use the M(ansible.windows.win_package) module instead.
options:
  name:
    description:
      - Package name, or package specifier with version.
      - Syntax varies with package manager. For example V(name-1.0) or V(name=1.0).
      - Package names also vary with package manager; this module will not "translate" them per distribution. For example V(libyaml-dev), V(libyaml-devel).
      - To operate on several packages this can accept a comma separated string of packages or a list of packages, depending on the underlying package manager.
    required: true
    type: raw
  state:
    description:
      - Whether to install (V(present)), or remove (V(absent)) a package.
      - You can use other states like V(latest) ONLY if they are supported by the underlying package module(s) executed.
    required: true
  use:
    description:
      - The required package manager module to use (V(dnf), V(apt), and so on). The default V(auto) will use existing facts or try to auto-detect it.
      - You should only use this field if the automatic selection is not working for some reason.
      - Since version 2.17 you can use the C(ansible_package_use) variable to override the automatic detection, but this option still takes precedence.
    default: auto
requirements:
    - Whatever is required for the package plugins specific for each system.
extends_documentation_fragment:
  -  action_common_attributes
  -  action_common_attributes.flow
attributes:
    action:
        support: full
    async:
        support: full
    bypass_host_loop:
        support: none
    check_mode:
        details: support depends on the underlying plugin invoked
        support: N/A
    diff_mode:
        details: support depends on the underlying plugin invoked
        support: N/A
    platform:
        details: The support depends on the availability for the specific plugin for each platform and if fact gathering is able to detect it
        platforms: all
notes:
    - While M(ansible.builtin.package) abstracts package managers to ease dealing with multiple distributions, package name often differs for the same software.

"""
EXAMPLES = """
- name: Install ntpdate
  ansible.builtin.package:
    name: ntpdate
    state: present

# This uses a variable as this changes per distribution.
- name: Remove the apache package
  ansible.builtin.package:
    name: "{{ apache }}"
    state: absent

- name: Install the latest version of Apache and MariaDB
  ansible.builtin.package:
    name:
      - httpd
      - mariadb-server
    state: latest
"""

import runpy

from ansible.module_utils.basic import AnsibleModule

from ansible.module_utils.common.text.converters import to_text
from ansible.module_utils.facts import ansible_collector
from ansible.module_utils.facts.collector import (
    CollectorNotFoundError,
    CycleFoundInFactDeps,
    UnresolvedFactDep,
)
from ansible.module_utils.facts.namespace import PrefixFactNamespace
from ansible.module_utils.facts.system.distribution import DistributionFactCollector
from ansible.module_utils.facts.system.pkg_mgr import PkgMgrFactCollector


def main():
    module = AnsibleModule(
        argument_spec=dict(
            name=dict(type="str"),
            state=dict(type="str"),
            use=dict(type="str", default="auto"),
        ),
        supports_check_mode=True,
    )

    p = module.params

    use = p["use"]

    if use == "auto":
        try:
            namespace = PrefixFactNamespace(namespace_name="ansible", prefix="ansible_")
            fact_collector = ansible_collector.AnsibleFactCollector(
                collectors=[
                    DistributionFactCollector(namespace=namespace),
                    PkgMgrFactCollector(namespace=namespace),
                ]
            )
            facts_dict = fact_collector.collect(module=module)
            use = facts_dict.get("ansible_pkg_mgr", None)
        except (
            TypeError,
            CollectorNotFoundError,
            CycleFoundInFactDeps,
            UnresolvedFactDep,
        ) as e:
            # bad subset given, collector, idk, deps declared but not found
            module.fail_json(msg=to_text(e))

    if use and use != "auto":
        try:
            runpy.run_module(
                mod_name=f"ansible.modules.{use}",
                run_name="__main__",
                alter_sys=True,
            )

            module.fail_json(
                msg=f'module "ansible.modules.{use}" did not exit successfully for unknown reasons. This is a bug.'
            )
        except ModuleNotFoundError:
            module.fail_json(msg=f'Package manager "{use}" is unsupported.')
        except Exception as e:
            module.fail_json(msg=f'Failed to delegate to "{use}" module: {e}')
    else:
        module.fail_json(
            msg='Could not detect which package manager to use. Try setting the "use" option.'
        )


if __name__ == "__main__":
    main()
