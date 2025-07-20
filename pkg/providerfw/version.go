// Copyright 2016-2025, Pulumi Corporation.
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

import (
	_ "embed"
	"strings"

	"github.com/blang/semver"

	"github.com/sapslaj/mid/version"
)

const (
	// FrameworkStateKeyName is the key name used to store the framework version in
	// a program's state.
	FrameworkStateKeyName = "pulumi-go-provider-version"
)

const VersionStr = version.Version

// Version is the framework's release version. Be aware that if the framework is installed
// from a git commit, this will refer to the next proposed release, not a
// format like "1.2.3-$COMMIT".
var FrameworkVersion semver.Version

func init() {
	FrameworkVersion = semver.MustParse(strings.TrimSpace(VersionStr))
}
