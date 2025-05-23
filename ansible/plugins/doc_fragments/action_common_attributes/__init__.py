# -*- coding: utf-8 -*-
# Copyright: Ansible Project
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)
from __future__ import annotations


class ModuleDocFragment(object):
    # Standard documentation fragment
    DOCUMENTATION = r"""
attributes:
    check_mode:
      description: Can run in check_mode and return changed status prediction without modifying target, if not supported the action will be skipped.
    diff_mode:
      description: Will return details on what has changed (or possibly needs changing in check_mode), when in diff mode
    platform:
      description: Target OS/families that can be operated against
      support: N/A
"""
