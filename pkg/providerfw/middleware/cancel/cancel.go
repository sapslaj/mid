// Copyright 2022, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cancel provides a middle-ware that ties the Cancel gRPC call from Pulumi to
// Go's [context.Context] cancellation system. See [Wrap].
package cancel

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/cancel/evict"
)

// Wrap ties a Pulumi's cancellation system to Go's [context.Context]'s cancellation
// system. It guarantees two things:
//
// 1. When a resource operation times out, the associated context is canceled.
//
// 2. When `Cancel` is called, all outstanding gRPC methods have their associated contexts
// canceled.
//
// A `Wrap`ped provider will still call the `Cancel` method on the underlying provider. If
// NotImplemented is returned, it will be swallowed.
func Wrap(provider p.Provider) p.Provider {
	cancelFuncs := evict.Pool[context.CancelFunc]{
		OnEvict: func(f context.CancelFunc) { f() },
	}
	cancel := func(ctx context.Context, timeout float64) (context.Context, func()) {
		var cancel context.CancelFunc
		if timeout == noTimeout {
			ctx, cancel = context.WithCancel(ctx)
		} else {
			ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeout))
		}

		handle := cancelFuncs.Insert(cancel)
		return ctx, handle.Evict
	}
	wrapper := provider
	wrapper.Cancel = func(ctx context.Context) error {
		cancelFuncs.Close()

		// We consider this a valid implementation of the Cancel RPC request. We still pass on
		// the request so downstream provides *may* rely on the Cancel call, but we catch an
		// Unimplemented error, making implementing the Cancel call optional for downstream
		// providers.
		var err error
		if provider.Cancel != nil {
			err = provider.Cancel(ctx)
			if status.Code(err) == codes.Unimplemented {
				return nil
			}
		}
		return err
	}

	// Wrap each gRPC method to transform a cancel call into a cancel on
	// context.Cancel.
	wrapper.GetSchema = setCancel2(cancel, provider.GetSchema, nil)
	wrapper.CheckConfig = setCancel2(cancel, provider.CheckConfig, nil)
	wrapper.DiffConfig = setCancel2(cancel, provider.DiffConfig, nil)
	wrapper.Configure = setCancel1(cancel, provider.Configure, nil)
	wrapper.Invoke = setCancel2(cancel, provider.Invoke, nil)
	wrapper.Check = setCancel2(cancel, provider.Check, nil)
	wrapper.Diff = setCancel2(cancel, provider.Diff, nil)
	wrapper.Create = setCancel2(cancel, provider.Create, func(r p.CreateRequest) float64 {
		return r.Timeout
	})
	wrapper.Read = setCancel2(cancel, provider.Read, nil)
	wrapper.Update = setCancel2(cancel, provider.Update, func(r p.UpdateRequest) float64 {
		return r.Timeout
	})
	wrapper.Delete = setCancel1(cancel, provider.Delete, func(r p.DeleteRequest) float64 {
		return r.Timeout
	})
	wrapper.Construct = setCancel2(cancel, provider.Construct, nil)
	wrapper.Call = setCancel2(cancel, provider.Call, nil)
	return wrapper
}

func setCancel1[
	Req any,
	F func(context.Context, Req) error,
	Cancel func(ctx context.Context, timeout float64) (context.Context, func()),
	GetTimeout func(Req) float64,
](cancel Cancel, f F, getTimeout GetTimeout) F {
	if f == nil {
		return nil
	}
	return func(ctx context.Context, req Req) error {
		var timeout float64
		if getTimeout != nil {
			timeout = getTimeout(req)
		}
		ctx, end := cancel(ctx, timeout)
		defer end()
		return f(ctx, req)
	}
}

func setCancel2[
	Req any, Resp any,
	F func(context.Context, Req) (Resp, error),
	Cancel func(ctx context.Context, timeout float64) (context.Context, func()),
	GetTimeout func(Req) float64,
](cancel Cancel, f F, getTimeout GetTimeout) F {
	if f == nil {
		return nil
	}
	return func(ctx context.Context, req Req) (Resp, error) {
		var timeout float64
		if getTimeout != nil {
			timeout = getTimeout(req)
		}

		ctx, end := cancel(ctx, timeout)
		defer end()
		return f(ctx, req)
	}
}

const noTimeout float64 = 0
