// Copyright 2025, Pulumi Corporation.
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

package integration

import (
	_ "unsafe" // unsafe is needed to use go:linkname

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	rpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	p "github.com/sapslaj/mid/pkg/providerfw"
)

type propertyToRPC func(m property.Map) (*structpb.Struct, error)

//go:linkname linkedConstructRequestToRPC github.com/sapslaj/mid/pkg/providerfw.linkedConstructRequestToRPC
func linkedConstructRequestToRPC(req *p.ConstructRequest, marshal propertyToRPC) *rpc.ConstructRequest

//go:linkname linkedConstructResponseFromRPC github.com/sapslaj/mid/pkg/providerfw.linkedConstructResponseFromRPC
func linkedConstructResponseFromRPC(resp *rpc.ConstructResponse) (p.ConstructResponse, error)

//go:linkname linkedCallRequestToRPC github.com/sapslaj/mid/pkg/providerfw.linkedCallRequestToRPC
func linkedCallRequestToRPC(req *p.CallRequest, marshal propertyToRPC) *rpc.CallRequest

//go:linkname linkedCallResponseFromRPC github.com/sapslaj/mid/pkg/providerfw.linkedCallResponseFromRPC
func linkedCallResponseFromRPC(resp *rpc.CallResponse) (p.CallResponse, error)
