import pulumi
import pulumi_ansible as ansible

vim = ansible.resource.Package("vim")
emacs = ansible.resource.Package("emacs",
    name="emacs",
    state="absent")
