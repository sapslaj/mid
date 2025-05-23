# -*- coding: utf-8 -*-
# Copyright: Ansible Project
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)
from __future__ import annotations


class ModuleDocFragment(object):
    DOCUMENTATION = r"""
attributes:
    action:
      details: While this action executes locally on the controller it is not governed by an action plugin
      support: none
    bypass_host_loop:
      support: none
    bypass_task_loop:
      support: none
    delegation:
      details: Since there are no connection nor facts, there is no sense in delegating includes
      support: none
    tags:
      details: Tags are interpreted by this action but are not automatically inherited by the include tasks, see C(apply)
      support: partial
"""
