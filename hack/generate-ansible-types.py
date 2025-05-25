#!/usr/bin/env python3
from io import StringIO
import json
import multiprocessing
import os
import pathlib
import re
import sys
from typing import Any
from importlib import import_module
import textwrap

import deepmerge
import yaml

agent_dir = pathlib.Path(__file__).parent / ".." / "agent"
ansible_dir = pathlib.Path(__file__).parent / ".." / "ansible"

sys.path.insert(0, str(ansible_dir.parent.absolute().resolve(True)))


def pascalcased(s: str) -> str:
    try:
        s = re.sub(r"['\"!@#$%^&\*\(\)\[\]\{\};:\,\./<>\?\|`~=\-+ ]+", "_", s)
    except Exception as e:
        print("EXCEPTION ON", s)
        raise e
    return "".join([word.capitalize() for word in s.split("_")])


def yaml_loads(data: Any) -> Any:
    yaml.safe_load(StringIO(data))
    return


def unmarkup(s: str) -> str:
    return re.sub(r"[A-Z]\((.+?)\)", r"`\1`", s)


def doc_comment(paragraphs: list[str] | str, indent: int) -> str:
    if isinstance(paragraphs, str):
        paragraphs = [paragraphs]
    indent_text = ("\t" * indent) + "// "
    result = ""
    for para in paragraphs:
        lines = textwrap.wrap(
            text=unmarkup(para),
            width=80,
            initial_indent=indent_text,
            subsequent_indent=indent_text,
        )
        for line in lines:
            result += line + "\n"
    return result.rstrip() + "\n"


def scalar_type_ansible_to_go(t: str) -> str:
    match t:
        case "str":
            return "string"
        case "path":
            return "string"
        case "bool":
            return "bool"
        case "int":
            return "int"
        case "raw":
            # TODO: should this be a string instead or something?
            return "any"
        case "float":
            return "float64"
        case "any":
            return "any"
        case _:
            raise Exception(f"unknown type '{t}'")


def composite_type_ansible_to_go(obj: Any) -> str:
    typ = obj.get("type", "str")
    match typ:
        case "complex":
            # TODO: handle this better
            return "any"
        case "list":
            elements = obj.get("elements", None)
            if not elements:
                return "[]any"
            if elements == "dict":
                suboptions = obj.get("suboptions", None)
                if not suboptions:
                    return "map[string]any"
                result = "struct {"
                for key, value in suboptions.items():
                    required = value.get("required", False)
                    result += "\t\t"
                    result += pascalcased(key)
                    result += " "
                    if not required:
                        result += "*"
                    result += composite_type_ansible_to_go(value)
                    result += ' `json:"'
                    result += key
                    if not required:
                        result += ",omitempty"
                    result += '"`\n'
                result += "\t}"
                return result
            else:
                return "[]" + scalar_type_ansible_to_go(elements)
        case "dict":
            elements = obj.get("elements", None)
            if elements is not None:
                match elements:
                    case "dict":
                        contains = obj.get("contains", None)
                        if contains is None:
                            raise Exception(
                                "dict has dict subelements but doesn't specify the type!"
                            )
                        result = "map[string]struct {"
                        for key, value in contains.items():
                            required = value.get("required", False)
                            result += "\t\t"
                            result += pascalcased(key)
                            result += " "
                            if not required:
                                result += "*"
                            result += composite_type_ansible_to_go(value)
                            result += ' `json:"'
                            result += key
                            if not required:
                                result += ",omitempty"
                            result += '"`\n'
                        result += "\t}"
                    case _:
                        raise Exception(f"dict has {elements} subelements???")
            suboptions = obj.get("suboptions", None)
            if not suboptions:
                return "map[string]any"
            result = "struct {"
            for key, value in suboptions.items():
                required = value.get("required", False)
                result += "\t\t"
                result += pascalcased(key)
                result += " "
                if not required:
                    result += "*"
                result += composite_type_ansible_to_go(value)
                result += ' `json:"'
                result += key
                if not required:
                    result += ",omitempty"
                result += '"`\n'

            result += "\t}"
            return result
        case _:
            return scalar_type_ansible_to_go(typ)


def process_module_file(module_file: str):
    try:
        if module_file.startswith("_"):
            return
        name = os.path.splitext(module_file)[0]
        print(name)

        module_fqn = f"ansible.modules.{name}"

        module = import_module(module_fqn)
        documentation = yaml.safe_load(StringIO(getattr(module, "DOCUMENTATION")))
        returns = yaml.safe_load(StringIO(getattr(module, "RETURN", "{}")))
        if returns is None:
            returns = dict()
        extends_documentation_fragments = documentation.get(
            "extends_documentation_fragment", []
        )
        if isinstance(extends_documentation_fragments, str):
            extends_documentation_fragments = [extends_documentation_fragments]
        for extends_documentation_fragment in extends_documentation_fragments:
            extends_documentation_fragment = (
                extends_documentation_fragment.removeprefix("ansible.builtin.")
            )
            try:
                doc_fragment_module = import_module(
                    f"ansible.plugins.doc_fragments.{extends_documentation_fragment}"
                )
                doc_fragment_class = getattr(
                    doc_fragment_module, "ModuleDocFragment", None
                )
                if doc_fragment_class is None:
                    continue
                doc_fragment = yaml.safe_load(
                    StringIO(getattr(doc_fragment_class, "DOCUMENTATION"))
                )
                documentation = deepmerge.always_merger.merge(
                    documentation, doc_fragment
                )
            except ModuleNotFoundError:
                parts = extends_documentation_fragment.split(".")
                subattr = parts[-1].upper()
                extends_documentation_fragment = ".".join(parts[:-1])
                doc_fragment_module = import_module(
                    f"ansible.plugins.doc_fragments.{extends_documentation_fragment}"
                )
                doc_fragment_class = getattr(
                    doc_fragment_module, "ModuleDocFragment", None
                )
                if doc_fragment_class is None:
                    continue
                doc_fragment = yaml.safe_load(
                    StringIO(getattr(doc_fragment_class, subattr))
                )
                documentation = deepmerge.always_merger.merge(
                    documentation, doc_fragment
                )

        pascalcase_name = pascalcased(name)

        with open(agent_dir / "ansible" / f"{name}.go", "w") as f:
            f.write(
                "// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT\n"
            )
            f.write("package ansible\n\n")
            f.write("import (\n")
            f.write('\t"github.com/sapslaj/mid/agent/rpc"\n')
            f.write(")\n\n")
            f.write(doc_comment(documentation["description"], indent=0))
            f.write(f'const {pascalcase_name}Name = "{name}"\n\n')
            for key, value in documentation["options"].items():
                if "choices" not in value:
                    continue
                if elements := value.get("elements", None):
                    enum_type = scalar_type_ansible_to_go(elements)
                else:
                    enum_type = scalar_type_ansible_to_go(value["type"])
                pascalcase_key = pascalcased(key)
                choicekeymap = value.get("__mid_codegen_choicekeymap", None)
                f.write(doc_comment(value["description"], indent=0))
                f.write(f"type {pascalcase_name}{pascalcase_key} {enum_type}\n\n")
                f.write("const (\n")
                for choice in value["choices"]:
                    if choicekeymap:
                        pascalcase_choice = choicekeymap[choice]
                    else:
                        pascalcase_choice = pascalcased(str(choice))
                    if isinstance(value["choices"], dict):
                        f.write(doc_comment(value["choices"][choice], indent=1))
                    f.write("\t")
                    f.write(f"{pascalcase_name}{pascalcase_key}{pascalcase_choice}")
                    f.write(" ")
                    f.write(f"{pascalcase_name}{pascalcase_key}")
                    f.write(" = ")
                    choice_repred = json.dumps(choice)
                    f.write(f"{choice_repred}\n")
                f.write(")\n\n")
                if not value.get("required", False):
                    f.write(f"func Optional{pascalcase_name}{pascalcase_key}")
                    f.write("[T interface {\n\t")
                    f.write(f"*{pascalcase_name}{pascalcase_key} | ")
                    f.write(f"{pascalcase_name}{pascalcase_key} | ")
                    f.write(f"*{enum_type} | {enum_type}")
                    f.write("\n}](s T) ")
                    f.write(f"*{pascalcase_name}{pascalcase_key}")
                    f.write(" {\n")
                    f.write("\tswitch v := any(s).(type) {\n")
                    f.write(f"\tcase *{pascalcase_name}{pascalcase_key}:\n")
                    f.write("\t\treturn v\n")
                    f.write(f"\tcase {pascalcase_name}{pascalcase_key}:\n")
                    f.write("\t\treturn &v\n")
                    f.write(f"\tcase *{enum_type}:\n")
                    f.write("\t\tif v == nil {\n")
                    f.write("\t\t\treturn nil\n")
                    f.write("\t\t}\n")
                    f.write(f"\t\tval := {pascalcase_name}{pascalcase_key}(*v)\n")
                    f.write("\t\treturn &val\n")
                    f.write(f"\tcase {enum_type}:\n")
                    f.write(f"\t\tval := {pascalcase_name}{pascalcase_key}(v)\n")
                    f.write("\t\treturn &val\n")
                    f.write("\tdefault:\n")
                    f.write('\t\tpanic("unsupported type")\n')
                    f.write("\t}\n")
                    f.write("}\n\n")
            f.write(
                doc_comment(f"Parameters for the `{name}` Ansible module.", indent=0)
            )
            f.write(f"type {pascalcase_name}Parameters struct {'{'}\n")
            for key, value in documentation["options"].items():
                required = value.get("required", False)
                f.write(doc_comment(value["description"], indent=1))
                if "default" in value:
                    default = value["default"]
                    default_repr = ""
                    if "choices" in value:
                        if value["type"] == "list":
                            default_repr = f"[]{pascalcase_name}{pascalcased(key)}"
                            default_repr += "{"
                            if not isinstance(default, list):
                                raise Exception(
                                    "list type does not use a list for the default value"
                                )
                            default_repr += ", ".join(
                                [
                                    f"{pascalcase_name}{pascalcased(key)}{pascalcased(default_choice)}"
                                    for default_choice in default
                                ]
                            )
                            default_repr += "}"
                        else:
                            default_repr = f"{pascalcase_name}{pascalcased(key)}{pascalcased(str(default))}"
                    elif default is None:
                        default_repr = "nil"
                    else:
                        default_repr = json.dumps(default)
                    f.write(doc_comment(f"default: {default_repr}", indent=1))
                f.write("\t")
                f.write(pascalcased(key))
                f.write(" ")
                if not required:
                    f.write("*")
                if "choices" in value:
                    f.write(f"{pascalcase_name}{pascalcased(key)}")
                else:
                    f.write(composite_type_ansible_to_go(value))
                f.write(' `json:"')
                f.write(key)
                if not required:
                    f.write(",omitempty")
                f.write('"`\n\n')
            f.write("}\n\n")
            f.write("")
            f.write(
                doc_comment(
                    f"Wrap the `{pascalcase_name}Parameters into an `rpc.RPCCall`.",
                    indent=0,
                )
            )
            f.write(
                f"func (p *{pascalcase_name}Parameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {'{'}\n"
            )
            f.write("\targs, err := rpc.AnyToJSONT[map[string]any](p)\n")
            f.write("\tif err != nil {\n")
            f.write("\t\treturn rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err\n")
            f.write("\t}\n")
            f.write("\treturn rpc.RPCCall[rpc.AnsibleExecuteArgs]{\n")
            f.write("\t\tRPCFunction: rpc.RPCAnsibleExecute,\n")
            f.write("\t\tArgs: rpc.AnsibleExecuteArgs{\n")
            f.write(f"\t\t\tName: {pascalcase_name}Name,\n")
            f.write("\t\t\tArgs: args,\n")
            f.write("\t\t},\n")
            f.write("\t}, nil\n")
            f.write("}\n\n")
            f.write(
                doc_comment(f"Return values for the `{name}` Ansible module.", indent=0)
            )
            f.write(f"type {pascalcase_name}Return struct {'{'}\n")
            f.write("\tAnsibleCommonReturns\n\n")
            for key, value in returns.items():
                f.write(doc_comment(value["description"], indent=1))
                f.write("\t")
                f.write(pascalcased(key))
                f.write(" *")
                f.write(composite_type_ansible_to_go(value))
                f.write(' `json:"')
                f.write(key)
                f.write(",omitempty")
                f.write('"`\n\n')
            f.write("}\n\n")
            f.write(
                doc_comment(
                    f"Unwrap the `rpc.RPCResult` into an `{pascalcase_name}Return`",
                    indent=0,
                )
            )
            f.write(
                f"func {pascalcase_name}ReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) ({pascalcase_name}Return, error) {'{'}\n"
            )
            f.write(
                f"\treturn rpc.AnyToJSONT[{pascalcase_name}Return](r.Result.Result)\n"
            )
            f.write("}\n")
    except Exception as e:
        raise Exception(f"Error while processing '{module_file}': {e}") from e


def main():
    [
        os.remove(agent_dir / "ansible" / generated)
        for generated in os.listdir(agent_dir / "ansible")
        if generated != "common.go"
    ]
    module_files = os.listdir(ansible_dir / "modules")
    with multiprocessing.Pool(os.process_cpu_count()) as p:
        p.map(process_module_file, module_files)


if __name__ == "__main__":
    main()
