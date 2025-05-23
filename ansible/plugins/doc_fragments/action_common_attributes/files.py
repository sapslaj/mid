# -*- coding: utf-8 -*-
# Copyright: Ansible Project
# GNU General Public License v3.0+ (see COPYING or https://www.gnu.org/licenses/gpl-3.0.txt)
from __future__ import annotations


class ModuleDocFragment(object):
    DOCUMENTATION = r"""
attributes:
    safe_file_operations:
      description: Uses Ansible's strict file operation functions to ensure proper permissions and avoid data corruption
    vault:
      description: Can automatically decrypt Ansible vaulted files
"""
