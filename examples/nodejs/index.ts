import * as pulumi from "@pulumi/pulumi";
import * as mid from "@sapslaj/pulumi-mid";

const provider = new mid.Provider("provider", {
  connection: {
    user: "root",
    password: "hunter2",
    host: "localhost",
    port: 2222,
  },
});
const vim = new mid.resource.Package("vim", {}, {
  provider: provider,
});
const emacs = new mid.resource.Package("emacs", {
  name: "emacs",
  ensure: "absent",
}, {
  provider: provider,
});
