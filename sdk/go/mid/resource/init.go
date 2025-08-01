// Code generated by pulumi-language-go DO NOT EDIT.
// *** WARNING: Do not edit by hand unless you're certain you know what you are doing! ***

package resource

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/mid/sdk/go/mid/internal"
)

type module struct {
	version semver.Version
}

func (m *module) Version() semver.Version {
	return m.version
}

func (m *module) Construct(ctx *pulumi.Context, name, typ, urn string) (r pulumi.Resource, err error) {
	switch typ {
	case "mid:resource:AnsibleTaskList":
		r = &AnsibleTaskList{}
	case "mid:resource:Apt":
		r = &Apt{}
	case "mid:resource:Exec":
		r = &Exec{}
	case "mid:resource:File":
		r = &File{}
	case "mid:resource:FileLine":
		r = &FileLine{}
	case "mid:resource:Group":
		r = &Group{}
	case "mid:resource:Package":
		r = &Package{}
	case "mid:resource:Service":
		r = &Service{}
	case "mid:resource:SystemdService":
		r = &SystemdService{}
	case "mid:resource:User":
		r = &User{}
	default:
		return nil, fmt.Errorf("unknown resource type: %s", typ)
	}

	err = ctx.RegisterResource(typ, name, nil, r, pulumi.URN_(urn))
	return
}

func init() {
	version, err := internal.PkgVersion()
	if err != nil {
		version = semver.Version{Major: 1}
	}
	pulumi.RegisterResourceModule(
		"mid",
		"resource",
		&module{version},
	)
}
