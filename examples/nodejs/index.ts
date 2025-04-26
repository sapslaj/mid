import * as pulumi from "@pulumi/pulumi";
import * as ansible from "@sapslaj/pulumi-provider-ansible";

const vim = new ansible.resource.Package("vim", {});
const emacs = new ansible.resource.Package("emacs", {
    name: "emacs",
    state: "absent",
});
