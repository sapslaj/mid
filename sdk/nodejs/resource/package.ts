// *** WARNING: this file was generated by pulumi-language-nodejs. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";
import * as inputs from "../types/input";
import * as outputs from "../types/output";
import * as utilities from "../utilities";

export class Package extends pulumi.CustomResource {
  /**
   * Get an existing Package resource's state with the given name, ID, and optional extra
   * properties used to qualify the lookup.
   *
   * @param name The _unique_ name of the resulting resource.
   * @param id The _unique_ provider ID of the resource to lookup.
   * @param opts Optional settings to control the behavior of the CustomResource.
   */
  public static get(name: string, id: pulumi.Input<pulumi.ID>, opts?: pulumi.CustomResourceOptions): Package {
    return new Package(name, undefined as any, { ...opts, id: id });
  }

  /** @internal */
  public static readonly __pulumiType = "mid:resource:Package";

  /**
   * Returns true if the given object is an instance of Package.  This is designed to work even
   * when multiple copies of the Pulumi SDK have been loaded into the same process.
   */
  public static isInstance(obj: any): obj is Package {
    if (obj === undefined || obj === null) {
      return false;
    }
    return obj["__pulumiType"] === Package.__pulumiType;
  }

  public readonly config!: pulumi.Output<outputs.ResourceConfig | undefined>;
  public readonly connection!: pulumi.Output<outputs.Connection | undefined>;
  public readonly ensure!: pulumi.Output<string>;
  public readonly name!: pulumi.Output<string | undefined>;
  public readonly names!: pulumi.Output<string[] | undefined>;
  public readonly triggers!: pulumi.Output<outputs.TriggersOutput>;

  /**
   * Create a Package resource with the given unique name, arguments, and options.
   *
   * @param name The _unique_ name of the resource.
   * @param args The arguments to use to populate this resource's properties.
   * @param opts A bag of options that control this resource's behavior.
   */
  constructor(name: string, args?: PackageArgs, opts?: pulumi.CustomResourceOptions) {
    let resourceInputs: pulumi.Inputs = {};
    opts = opts || {};
    if (!opts.id) {
      resourceInputs["config"] = args ? args.config : undefined;
      resourceInputs["connection"] = args
        ? (args.connection ? pulumi.output(args.connection).apply(inputs.connectionArgsProvideDefaults) : undefined)
        : undefined;
      resourceInputs["ensure"] = args ? args.ensure : undefined;
      resourceInputs["name"] = args ? args.name : undefined;
      resourceInputs["names"] = args ? args.names : undefined;
      resourceInputs["triggers"] = args ? args.triggers : undefined;
    } else {
      resourceInputs["config"] = undefined /*out*/;
      resourceInputs["connection"] = undefined /*out*/;
      resourceInputs["ensure"] = undefined /*out*/;
      resourceInputs["name"] = undefined /*out*/;
      resourceInputs["names"] = undefined /*out*/;
      resourceInputs["triggers"] = undefined /*out*/;
    }
    opts = pulumi.mergeOptions(utilities.resourceOptsDefaults(), opts);
    super(Package.__pulumiType, name, resourceInputs, opts);
  }
}

/**
 * The set of arguments for constructing a Package resource.
 */
export interface PackageArgs {
  config?: pulumi.Input<inputs.ResourceConfigArgs>;
  connection?: pulumi.Input<inputs.ConnectionArgs>;
  ensure?: pulumi.Input<string>;
  name?: pulumi.Input<string>;
  names?: pulumi.Input<pulumi.Input<string>[]>;
  triggers?: pulumi.Input<inputs.TriggersInputArgs>;
}
