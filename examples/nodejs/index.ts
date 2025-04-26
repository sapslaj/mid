import * as pulumi from "@pulumi/pulumi";
import * as mid from "@pulumi/mid";

const myRandomResource = new mid.Random("myRandomResource", {length: 24});
export const output = {
    value: myRandomResource.result,
};
