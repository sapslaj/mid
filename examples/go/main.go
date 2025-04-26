package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/sapslaj/mid/sdk/go/mid"
	"github.com/sapslaj/mid/sdk/go/mid/resource"
	"github.com/sapslaj/mid/sdk/go/mid/types"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		provider, err := mid.NewProvider(ctx, "provider", &mid.ProviderArgs{
			Connection: &types.ConnectionArgs{
				User:     pulumi.String("root"),
				Password: pulumi.String("hunter2"),
				Host:     pulumi.String("localhost"),
				Port:     pulumi.Float64(2222),
			},
		})
		if err != nil {
			return err
		}
		_, err = resource.NewPackage(ctx, "vim", nil, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		_, err = resource.NewPackage(ctx, "emacs", &resource.PackageArgs{
			Name:  pulumi.String("emacs"),
			State: pulumi.String("absent"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		return nil
	})
}
