// *** WARNING: this file was generated by pulumi-language-nodejs. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";
import * as inputs from "../types/input";
import * as outputs from "../types/output";

export namespace agent {
}

export namespace resource {
}

export namespace types {
  export interface ConnectionArgs {
    host: pulumi.Input<string>;
    password?: pulumi.Input<string>;
    port?: pulumi.Input<number>;
    privateKey?: pulumi.Input<string>;
    user?: pulumi.Input<string>;
  }

  export interface ExecCommandArgs {
    command: pulumi.Input<pulumi.Input<string>[]>;
    dir?: pulumi.Input<string>;
    environment?: pulumi.Input<{ [key: string]: pulumi.Input<string> }>;
    stdin?: pulumi.Input<string>;
  }

  export interface TriggersInputArgs {
    refresh?: pulumi.Input<any[]>;
    replace?: pulumi.Input<any[]>;
  }
}
