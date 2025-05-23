# -*- coding: utf-8 -*-
# Copyright: Ansible Project
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)
from __future__ import annotations


class ModuleDocFragment(object):
    DOCUMENTATION = r"""
attributes:
    action:
      description: Indicates this has a corresponding action plugin so some parts of the options can be executed on the controller
    async:
      description: Supports being used with the C(async) keyword
    bypass_host_loop:
      description:
            - Forces a 'global' task that does not execute per host, this bypasses per host templating and serial,
              throttle and other loop considerations
            - Conditionals will work as if C(run_once) is being used, variables used will be from the first available host
            - This action will not work normally outside of lockstep strategies
"""
