package types

type TriggersInput struct {
	Refresh *[]any `pulumi:"refresh,optional"`
	Replace *[]any `pulumi:"replace,optional" provider:"replaceOnChanges"`
}

type TriggersOutput struct {
	Refresh     *[]any `pulumi:"refresh,optional"`
	Replace     *[]any `pulumi:"replace,optional"`
	LastChanged string `pulumi:"lastChanged"`
}
