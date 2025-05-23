# -*- coding: utf-8 -*-
# Copyright: Ansible Project
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)
from __future__ import annotations


class ModuleDocFragment(object):
    DOCUMENTATION = r"""
attributes:
    become:
      description: Is usable alongside become keywords
    connection:
      description: Uses the target's configured connection information to execute code on it
    delegation:
      description: Can be used in conjunction with delegate_to and related keywords
"""
