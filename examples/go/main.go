package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/pulumi-provider-ansible/sdk/go/ansible/resource"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := resource.NewPackage(ctx, "vim", nil)
		if err != nil {
			return err
		}
		_, err = resource.NewPackage(ctx, "emacs", &resource.PackageArgs{
			Name:  pulumi.String("emacs"),
			State: pulumi.String("absent"),
		})
		if err != nil {
			return err
		}
		return nil
	})
}
