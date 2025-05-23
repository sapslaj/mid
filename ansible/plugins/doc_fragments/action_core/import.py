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
      details: While the import can be host specific and runs per host it is not dealing with all available host variables,
               use an include instead for those cases
      support: partial
    bypass_task_loop:
      details: The task itself is not looped, but the loop is applied to each imported task
      support: partial
    delegation:
      details: Since there are no connection nor facts, there is no sense in delegating imports
      support: none
    ignore_conditional:
      details: While the action itself will ignore the conditional, it will be inherited by the imported tasks themselves
      support: partial
    tags:
      details: Tags are not interpreted for this action, they are applied to the imported tasks
      support: none
    until:
      support: none
"""
