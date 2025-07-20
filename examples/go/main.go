package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/mid/sdk/go/mid"
	"github.com/sapslaj/mid/sdk/go/mid/resource"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		provider, err := mid.NewProvider(ctx, "provider", &mid.ProviderArgs{
			Connection: &mid.ConnectionArgs{
				User:     pulumi.String("root"),
				Password: pulumi.String("hunter2"),
				Host:     pulumi.String("localhost"),
				Port:     pulumi.Float64(2222),
			},
		})
		if err != nil {
			return err
		}
		_, err = resource.NewPackage(ctx, "vim", &resource.PackageArgs{
			Name:   pulumi.String("vim"),
			Ensure: pulumi.String("present"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		_, err = resource.NewPackage(ctx, "emacs", &resource.PackageArgs{
			Name:   pulumi.String("emacs"),
			Ensure: pulumi.String("absent"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		return nil
	})
}
