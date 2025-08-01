// Code generated by pulumi-language-go DO NOT EDIT.
// *** WARNING: Do not edit by hand unless you're certain you know what you are doing! ***

package resource

import (
	"context"
	"reflect"

	"errors"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/mid/sdk/go/mid"
	"github.com/sapslaj/mid/sdk/go/mid/internal"
)

type AnsibleTaskList struct {
	pulumi.CustomResourceState

	Config     mid.ResourceConfigPtrOutput       `pulumi:"config"`
	Connection mid.ConnectionPtrOutput           `pulumi:"connection"`
	Results    AnsibleTaskListStateResultsOutput `pulumi:"results"`
	Tasks      AnsibleTaskListArgsTasksOutput    `pulumi:"tasks"`
	Triggers   mid.TriggersOutputOutput          `pulumi:"triggers"`
}

// NewAnsibleTaskList registers a new resource with the given unique name, arguments, and options.
func NewAnsibleTaskList(ctx *pulumi.Context,
	name string, args *AnsibleTaskListArgs, opts ...pulumi.ResourceOption) (*AnsibleTaskList, error) {
	if args == nil {
		return nil, errors.New("missing one or more required arguments")
	}

	if args.Tasks == nil {
		return nil, errors.New("invalid value for required argument 'Tasks'")
	}
	if args.Connection != nil {
		args.Connection = args.Connection.ToConnectionPtrOutput().ApplyT(func(v *mid.Connection) *mid.Connection { return v.Defaults() }).(mid.ConnectionPtrOutput)
	}
	opts = internal.PkgResourceDefaultOpts(opts)
	var resource AnsibleTaskList
	err := ctx.RegisterResource("mid:resource:AnsibleTaskList", name, args, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetAnsibleTaskList gets an existing AnsibleTaskList resource's state with the given name, ID, and optional
// state properties that are used to uniquely qualify the lookup (nil if not required).
func GetAnsibleTaskList(ctx *pulumi.Context,
	name string, id pulumi.IDInput, state *AnsibleTaskListState, opts ...pulumi.ResourceOption) (*AnsibleTaskList, error) {
	var resource AnsibleTaskList
	err := ctx.ReadResource("mid:resource:AnsibleTaskList", name, id, state, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// Input properties used for looking up and filtering AnsibleTaskList resources.
type ansibleTaskListState struct {
}

type AnsibleTaskListState struct {
}

func (AnsibleTaskListState) ElementType() reflect.Type {
	return reflect.TypeOf((*ansibleTaskListState)(nil)).Elem()
}

type ansibleTaskListArgs struct {
	Config     *mid.ResourceConfig      `pulumi:"config"`
	Connection *mid.Connection          `pulumi:"connection"`
	Tasks      AnsibleTaskListArgsTasks `pulumi:"tasks"`
	Triggers   *mid.TriggersInput       `pulumi:"triggers"`
}

// The set of arguments for constructing a AnsibleTaskList resource.
type AnsibleTaskListArgs struct {
	Config     mid.ResourceConfigPtrInput
	Connection mid.ConnectionPtrInput
	Tasks      AnsibleTaskListArgsTasksInput
	Triggers   mid.TriggersInputPtrInput
}

func (AnsibleTaskListArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*ansibleTaskListArgs)(nil)).Elem()
}

type AnsibleTaskListInput interface {
	pulumi.Input

	ToAnsibleTaskListOutput() AnsibleTaskListOutput
	ToAnsibleTaskListOutputWithContext(ctx context.Context) AnsibleTaskListOutput
}

func (*AnsibleTaskList) ElementType() reflect.Type {
	return reflect.TypeOf((**AnsibleTaskList)(nil)).Elem()
}

func (i *AnsibleTaskList) ToAnsibleTaskListOutput() AnsibleTaskListOutput {
	return i.ToAnsibleTaskListOutputWithContext(context.Background())
}

func (i *AnsibleTaskList) ToAnsibleTaskListOutputWithContext(ctx context.Context) AnsibleTaskListOutput {
	return pulumi.ToOutputWithContext(ctx, i).(AnsibleTaskListOutput)
}

// AnsibleTaskListArrayInput is an input type that accepts AnsibleTaskListArray and AnsibleTaskListArrayOutput values.
// You can construct a concrete instance of `AnsibleTaskListArrayInput` via:
//
//	AnsibleTaskListArray{ AnsibleTaskListArgs{...} }
type AnsibleTaskListArrayInput interface {
	pulumi.Input

	ToAnsibleTaskListArrayOutput() AnsibleTaskListArrayOutput
	ToAnsibleTaskListArrayOutputWithContext(context.Context) AnsibleTaskListArrayOutput
}

type AnsibleTaskListArray []AnsibleTaskListInput

func (AnsibleTaskListArray) ElementType() reflect.Type {
	return reflect.TypeOf((*[]*AnsibleTaskList)(nil)).Elem()
}

func (i AnsibleTaskListArray) ToAnsibleTaskListArrayOutput() AnsibleTaskListArrayOutput {
	return i.ToAnsibleTaskListArrayOutputWithContext(context.Background())
}

func (i AnsibleTaskListArray) ToAnsibleTaskListArrayOutputWithContext(ctx context.Context) AnsibleTaskListArrayOutput {
	return pulumi.ToOutputWithContext(ctx, i).(AnsibleTaskListArrayOutput)
}

// AnsibleTaskListMapInput is an input type that accepts AnsibleTaskListMap and AnsibleTaskListMapOutput values.
// You can construct a concrete instance of `AnsibleTaskListMapInput` via:
//
//	AnsibleTaskListMap{ "key": AnsibleTaskListArgs{...} }
type AnsibleTaskListMapInput interface {
	pulumi.Input

	ToAnsibleTaskListMapOutput() AnsibleTaskListMapOutput
	ToAnsibleTaskListMapOutputWithContext(context.Context) AnsibleTaskListMapOutput
}

type AnsibleTaskListMap map[string]AnsibleTaskListInput

func (AnsibleTaskListMap) ElementType() reflect.Type {
	return reflect.TypeOf((*map[string]*AnsibleTaskList)(nil)).Elem()
}

func (i AnsibleTaskListMap) ToAnsibleTaskListMapOutput() AnsibleTaskListMapOutput {
	return i.ToAnsibleTaskListMapOutputWithContext(context.Background())
}

func (i AnsibleTaskListMap) ToAnsibleTaskListMapOutputWithContext(ctx context.Context) AnsibleTaskListMapOutput {
	return pulumi.ToOutputWithContext(ctx, i).(AnsibleTaskListMapOutput)
}

type AnsibleTaskListOutput struct{ *pulumi.OutputState }

func (AnsibleTaskListOutput) ElementType() reflect.Type {
	return reflect.TypeOf((**AnsibleTaskList)(nil)).Elem()
}

func (o AnsibleTaskListOutput) ToAnsibleTaskListOutput() AnsibleTaskListOutput {
	return o
}

func (o AnsibleTaskListOutput) ToAnsibleTaskListOutputWithContext(ctx context.Context) AnsibleTaskListOutput {
	return o
}

func (o AnsibleTaskListOutput) Config() mid.ResourceConfigPtrOutput {
	return o.ApplyT(func(v *AnsibleTaskList) mid.ResourceConfigPtrOutput { return v.Config }).(mid.ResourceConfigPtrOutput)
}

func (o AnsibleTaskListOutput) Connection() mid.ConnectionPtrOutput {
	return o.ApplyT(func(v *AnsibleTaskList) mid.ConnectionPtrOutput { return v.Connection }).(mid.ConnectionPtrOutput)
}

func (o AnsibleTaskListOutput) Results() AnsibleTaskListStateResultsOutput {
	return o.ApplyT(func(v *AnsibleTaskList) AnsibleTaskListStateResultsOutput { return v.Results }).(AnsibleTaskListStateResultsOutput)
}

func (o AnsibleTaskListOutput) Tasks() AnsibleTaskListArgsTasksOutput {
	return o.ApplyT(func(v *AnsibleTaskList) AnsibleTaskListArgsTasksOutput { return v.Tasks }).(AnsibleTaskListArgsTasksOutput)
}

func (o AnsibleTaskListOutput) Triggers() mid.TriggersOutputOutput {
	return o.ApplyT(func(v *AnsibleTaskList) mid.TriggersOutputOutput { return v.Triggers }).(mid.TriggersOutputOutput)
}

type AnsibleTaskListArrayOutput struct{ *pulumi.OutputState }

func (AnsibleTaskListArrayOutput) ElementType() reflect.Type {
	return reflect.TypeOf((*[]*AnsibleTaskList)(nil)).Elem()
}

func (o AnsibleTaskListArrayOutput) ToAnsibleTaskListArrayOutput() AnsibleTaskListArrayOutput {
	return o
}

func (o AnsibleTaskListArrayOutput) ToAnsibleTaskListArrayOutputWithContext(ctx context.Context) AnsibleTaskListArrayOutput {
	return o
}

func (o AnsibleTaskListArrayOutput) Index(i pulumi.IntInput) AnsibleTaskListOutput {
	return pulumi.All(o, i).ApplyT(func(vs []interface{}) *AnsibleTaskList {
		return vs[0].([]*AnsibleTaskList)[vs[1].(int)]
	}).(AnsibleTaskListOutput)
}

type AnsibleTaskListMapOutput struct{ *pulumi.OutputState }

func (AnsibleTaskListMapOutput) ElementType() reflect.Type {
	return reflect.TypeOf((*map[string]*AnsibleTaskList)(nil)).Elem()
}

func (o AnsibleTaskListMapOutput) ToAnsibleTaskListMapOutput() AnsibleTaskListMapOutput {
	return o
}

func (o AnsibleTaskListMapOutput) ToAnsibleTaskListMapOutputWithContext(ctx context.Context) AnsibleTaskListMapOutput {
	return o
}

func (o AnsibleTaskListMapOutput) MapIndex(k pulumi.StringInput) AnsibleTaskListOutput {
	return pulumi.All(o, k).ApplyT(func(vs []interface{}) *AnsibleTaskList {
		return vs[0].(map[string]*AnsibleTaskList)[vs[1].(string)]
	}).(AnsibleTaskListOutput)
}

func init() {
	pulumi.RegisterInputType(reflect.TypeOf((*AnsibleTaskListInput)(nil)).Elem(), &AnsibleTaskList{})
	pulumi.RegisterInputType(reflect.TypeOf((*AnsibleTaskListArrayInput)(nil)).Elem(), AnsibleTaskListArray{})
	pulumi.RegisterInputType(reflect.TypeOf((*AnsibleTaskListMapInput)(nil)).Elem(), AnsibleTaskListMap{})
	pulumi.RegisterOutputType(AnsibleTaskListOutput{})
	pulumi.RegisterOutputType(AnsibleTaskListArrayOutput{})
	pulumi.RegisterOutputType(AnsibleTaskListMapOutput{})
}
