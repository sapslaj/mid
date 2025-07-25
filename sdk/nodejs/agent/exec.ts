// *** WARNING: this file was generated by pulumi-language-nodejs. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";
import * as inputs from "../types/input";
import * as outputs from "../types/output";
import * as utilities from "../utilities";

export function exec(args: ExecArgs, opts?: pulumi.InvokeOptions): Promise<ExecResult> {
  opts = pulumi.mergeOptions(utilities.resourceOptsDefaults(), opts || {});
  return pulumi.runtime.invoke("mid:agent:exec", {
    "command": args.command,
    "config": args.config,
    "connection": args.connection ? inputs.connectionProvideDefaults(args.connection) : undefined,
    "dir": args.dir,
    "environment": args.environment,
    "expandArgumentVars": args.expandArgumentVars,
    "stdin": args.stdin,
  }, opts);
}

export interface ExecArgs {
  command: string[];
  config?: inputs.ResourceConfig;
  connection?: inputs.Connection;
  dir?: string;
  environment?: { [key: string]: string };
  expandArgumentVars?: boolean;
  stdin?: string;
}

export interface ExecResult {
  readonly command: string[];
  readonly config?: outputs.ResourceConfig;
  readonly connection?: outputs.Connection;
  readonly dir?: string;
  readonly environment?: { [key: string]: string };
  readonly exitCode: number;
  readonly expandArgumentVars?: boolean;
  readonly pid: number;
  readonly stderr: string;
  readonly stdin?: string;
  readonly stdout: string;
}
export function execOutput(args: ExecOutputArgs, opts?: pulumi.InvokeOutputOptions): pulumi.Output<ExecResult> {
  opts = pulumi.mergeOptions(utilities.resourceOptsDefaults(), opts || {});
  return pulumi.runtime.invokeOutput("mid:agent:exec", {
    "command": args.command,
    "config": args.config,
    "connection": args.connection ? pulumi.output(args.connection).apply(inputs.connectionProvideDefaults) : undefined,
    "dir": args.dir,
    "environment": args.environment,
    "expandArgumentVars": args.expandArgumentVars,
    "stdin": args.stdin,
  }, opts);
}

export interface ExecOutputArgs {
  command: pulumi.Input<pulumi.Input<string>[]>;
  config?: pulumi.Input<inputs.ResourceConfigArgs>;
  connection?: pulumi.Input<inputs.ConnectionArgs>;
  dir?: pulumi.Input<string>;
  environment?: pulumi.Input<{ [key: string]: pulumi.Input<string> }>;
  expandArgumentVars?: pulumi.Input<boolean>;
  stdin?: pulumi.Input<string>;
}
