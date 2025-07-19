package midtypes

import "github.com/pulumi/pulumi-go-provider/infer"

type TriggersInput struct {
	Refresh *[]any `pulumi:"refresh,optional"`
	Replace *[]any `pulumi:"replace,optional" provider:"replaceOnChanges"`
}

func (i *TriggersInput) Annotate(a infer.Annotator) {
	a.Describe(
		&i.Refresh,
		`Run any "refresh" operations (e.g. service restarts, change diffs, etc) if
any value in this list changes.`,
	)
	a.Describe(
		&i.Replace,
		`Completely delete and replace the resource if any value in this list
changes.`,
	)
}

type TriggersOutput struct {
	Refresh     *[]any `pulumi:"refresh,optional"`
	Replace     *[]any `pulumi:"replace,optional"`
	LastChanged string `pulumi:"lastChanged"`
}

func (i *TriggersOutput) Annotate(a infer.Annotator) {
	a.Describe(
		&i.Refresh,
		`Run any "refresh" operations (e.g. service restarts, change diffs, etc) if
any value in this list changes.`,
	)
	a.Describe(
		&i.Replace,
		`Completely delete and replace the resource if any value in this list
changes.`,
	)
	a.Describe(
		&i.LastChanged,
		`RFC 3339 timestamp of when this resource last changed. Use this property
to chain into other resources' `+"`refresh` and `replace`"+` triggers.`,
	)
}
