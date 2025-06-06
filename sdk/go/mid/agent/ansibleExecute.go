// Code generated by pulumi-language-go DO NOT EDIT.
// *** WARNING: Do not edit by hand unless you're certain you know what you are doing! ***

package agent

import (
	"context"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/mid/sdk/go/mid/internal"
)

func AnsibleExecute(ctx *pulumi.Context, args *AnsibleExecuteArgs, opts ...pulumi.InvokeOption) (*AnsibleExecuteResult, error) {
	opts = internal.PkgInvokeDefaultOpts(opts)
	var rv AnsibleExecuteResult
	err := ctx.Invoke("mid:agent:ansibleExecute", args, &rv, opts...)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

type AnsibleExecuteArgs struct {
	Args               map[string]interface{} `pulumi:"args"`
	Check              *bool                  `pulumi:"check"`
	DebugKeepTempFiles *bool                  `pulumi:"debugKeepTempFiles"`
	Environment        map[string]string      `pulumi:"environment"`
	Name               string                 `pulumi:"name"`
}

type AnsibleExecuteResult struct {
	Args               map[string]interface{} `pulumi:"args"`
	Check              *bool                  `pulumi:"check"`
	DebugKeepTempFiles *bool                  `pulumi:"debugKeepTempFiles"`
	DebugTempDir       *string                `pulumi:"debugTempDir"`
	Environment        map[string]string      `pulumi:"environment"`
	ExitCode           int                    `pulumi:"exitCode"`
	Name               string                 `pulumi:"name"`
	Result             map[string]interface{} `pulumi:"result"`
	Stderr             string                 `pulumi:"stderr"`
	Stdout             string                 `pulumi:"stdout"`
}

func AnsibleExecuteOutput(ctx *pulumi.Context, args AnsibleExecuteOutputArgs, opts ...pulumi.InvokeOption) AnsibleExecuteResultOutput {
	return pulumi.ToOutputWithContext(ctx.Context(), args).
		ApplyT(func(v interface{}) (AnsibleExecuteResultOutput, error) {
			args := v.(AnsibleExecuteArgs)
			options := pulumi.InvokeOutputOptions{InvokeOptions: internal.PkgInvokeDefaultOpts(opts)}
			return ctx.InvokeOutput("mid:agent:ansibleExecute", args, AnsibleExecuteResultOutput{}, options).(AnsibleExecuteResultOutput), nil
		}).(AnsibleExecuteResultOutput)
}

type AnsibleExecuteOutputArgs struct {
	Args               pulumi.MapInput       `pulumi:"args"`
	Check              pulumi.BoolPtrInput   `pulumi:"check"`
	DebugKeepTempFiles pulumi.BoolPtrInput   `pulumi:"debugKeepTempFiles"`
	Environment        pulumi.StringMapInput `pulumi:"environment"`
	Name               pulumi.StringInput    `pulumi:"name"`
}

func (AnsibleExecuteOutputArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*AnsibleExecuteArgs)(nil)).Elem()
}

type AnsibleExecuteResultOutput struct{ *pulumi.OutputState }

func (AnsibleExecuteResultOutput) ElementType() reflect.Type {
	return reflect.TypeOf((*AnsibleExecuteResult)(nil)).Elem()
}

func (o AnsibleExecuteResultOutput) ToAnsibleExecuteResultOutput() AnsibleExecuteResultOutput {
	return o
}

func (o AnsibleExecuteResultOutput) ToAnsibleExecuteResultOutputWithContext(ctx context.Context) AnsibleExecuteResultOutput {
	return o
}

func (o AnsibleExecuteResultOutput) Args() pulumi.MapOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) map[string]interface{} { return v.Args }).(pulumi.MapOutput)
}

func (o AnsibleExecuteResultOutput) Check() pulumi.BoolPtrOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) *bool { return v.Check }).(pulumi.BoolPtrOutput)
}

func (o AnsibleExecuteResultOutput) DebugKeepTempFiles() pulumi.BoolPtrOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) *bool { return v.DebugKeepTempFiles }).(pulumi.BoolPtrOutput)
}

func (o AnsibleExecuteResultOutput) DebugTempDir() pulumi.StringPtrOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) *string { return v.DebugTempDir }).(pulumi.StringPtrOutput)
}

func (o AnsibleExecuteResultOutput) Environment() pulumi.StringMapOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) map[string]string { return v.Environment }).(pulumi.StringMapOutput)
}

func (o AnsibleExecuteResultOutput) ExitCode() pulumi.IntOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) int { return v.ExitCode }).(pulumi.IntOutput)
}

func (o AnsibleExecuteResultOutput) Name() pulumi.StringOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) string { return v.Name }).(pulumi.StringOutput)
}

func (o AnsibleExecuteResultOutput) Result() pulumi.MapOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) map[string]interface{} { return v.Result }).(pulumi.MapOutput)
}

func (o AnsibleExecuteResultOutput) Stderr() pulumi.StringOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) string { return v.Stderr }).(pulumi.StringOutput)
}

func (o AnsibleExecuteResultOutput) Stdout() pulumi.StringOutput {
	return o.ApplyT(func(v AnsibleExecuteResult) string { return v.Stdout }).(pulumi.StringOutput)
}

func init() {
	pulumi.RegisterOutputType(AnsibleExecuteResultOutput{})
}
