// Copyright 2024, Pulumi Corporation.
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

package providerfw

import "fmt"

// Error indicates a bug in the pulumi-go-provider framework.
type Error struct {
	Inner error
}

// Errorf creates a new [Error] by delegating to [fmt.Errorf].
func Errorf(msg string, a ...any) error {
	return Error{fmt.Errorf(msg, a...)}
}

func (err Error) Error() string {
	const (
		prefix = "internal error"
		suffix = "; please report this to https://github.com/sapslaj/mid/pkg/providerfw/issues"
	)
	if err.Inner == nil {
		return prefix + suffix
	}
	return prefix + ": " + err.Inner.Error() + suffix
}
