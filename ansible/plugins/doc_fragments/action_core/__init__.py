# -*- coding: utf-8 -*-
# Copyright: (c) , Ansible Project
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)
from __future__ import annotations


# WARNING: this is mostly here as a convenience for documenting core behaviours, no plugin outside of ansible-core should use this file
class ModuleDocFragment(object):
    # requires action_common
    DOCUMENTATION = r"""
attributes:
    async:
      support: none
    become:
      support: none
    bypass_task_loop:
      description: These tasks ignore the C(loop) and C(with_) keywords
    core:
      description: This is a 'core engine' feature and is not implemented like most task actions, so it is not overridable in any way via the plugin system.
      support: full
    connection:
      support: none
    ignore_conditional:
      support: none
      description: The action is not subject to conditional execution so it will ignore the C(when:) keyword
    platform:
      support: full
      platforms: all
    until:
      description: Denotes if this action obeys until/retry/poll keywords
      support: full
    tags:
      description: Allows for the 'tags' keyword to control the selection of this action for execution
      support: full
"""
