#!/usr/bin/env -S just --justfile

provider_version := `git describe --tags --abbrev=0` + "-alpha.0+dev"
testparallelism := "4"

[group('sdk')]
[group('provider')]
build: provider sdk-go sdk-nodejs sdk-python

[group('sdk')]
[group('provider')]
build-debug: provider-debug sdk-go sdk-nodejs sdk-python

[group('sdk')]
[group('provider')]
install: install-provider install-nodejs-sdk

fmt:
  # TODO: figure out how to run `go fmt` via dprint
  go fmt ./...
  `pwd`/bin/goimports -w **/*.go
  `pwd`/bin/gci write --custom-order -s standard -s default -s 'prefix(github.com/sapslaj/mid)' **/*.go
  dprint fmt
  -pre-commit run -a

[group('examples')]
[group('tidy')]
tidy-examples:
  cd examples/go && go mod tidy

[group('provider')]
[group('tidy')]
tidy-provider:
  cd provider && go mod tidy

[group('sdk')]
[group('tidy')]
tidy-sdk:
  cd sdk && go mod tidy

[group('tidy')]
tidy: tidy-examples tidy-provider tidy-sdk

set-version:
  ./hack/set-version.sh '{{ provider_version }}'

[group('agent')]
agent-ansible-bundle:
  tar \
    --create \
    --numeric-owner \
    --owner 0 \
    --group 0 \
    --no-same-owner \
    --no-same-permissions \
    --gzip \
    --file ./agent/cmd/mid-agent/ansible.tar.gz \
    --exclude-ignore=./.gitignore \
    ./ansible

[group('agent')]
agent-codegen: agent-ansible-bundle
  rm -f agent/mid-agent-*
  uv run ./hack/generate-ansible-types.py
  uv run ./hack/generate-agent-binaries.py
  go fmt ./...

[group('provider')]
provider: set-version agent-codegen
  go build \
    -o ./bin/pulumi-resource-mid \
    "github.com/sapslaj/mid/provider/cmd/pulumi-resource-mid"

[group('provider')]
provider-debug: set-version agent-codegen
  go build \
    -o ./bin/pulumi-resource-mid \
    -cover \
    -covermode=set \
    -coverpkg=github.com/sapslaj/mid/agent,github.com/sapslaj/mid/provider/...,github.com/sapslaj/mid/pkg... \
    -gcflags="all=-N -l" \
    "github.com/sapslaj/mid/provider/cmd/pulumi-resource-mid"

[group('provider')]
[group('test')]
test-provider:
  cd tests && go test -short -v -count=1 -cover -timeout 2h -parallel {{ testparallelism }} ./...

[group('sdk')]
sdk-go: provider
  #!/usr/bin/env sh
  set -eu
  rm -rf sdk/go
  pulumi package gen-sdk ./bin/pulumi-resource-mid --language go

[group('sdk')]
sdk-nodejs: provider
  #!/usr/bin/env sh
  set -eu
  rm -rf sdk/nodejs
  pulumi package gen-sdk ./bin/pulumi-resource-mid --language nodejs
  cd sdk/nodejs
  npm install
  npm run build
  cp ../../README.md ../../LICENSE package.json package-lock.json bin/

[group('sdk')]
sdk-python: provider
  #!/usr/bin/env sh
  set -eu
  rm -rf sdk/python
  pulumi package gen-sdk ./bin/pulumi-resource-mid --language python
  cp ./README.md ./sdk/python/

[group('sdk')]
sdk: sdk-go sdk-nodejs sdk-python

[group('examples')]
examples-go:
  rm -rf ./examples/go
  pulumi convert --cwd ./examples/yaml --logtostderr --generate-only --non-interactive --language go --out `pwd`/examples/go

[group('examples')]
examples-nodejs:
  rm -rf ./examples/nodejs
  pulumi convert --cwd ./examples/yaml --logtostderr --generate-only --non-interactive --language nodejs --out `pwd`/examples/nodejs

[group('examples')]
examples-python:
  rm -rf ./examples/python
  pulumi convert --cwd ./examples/yaml --logtostderr --generate-only --non-interactive --language python --out `pwd`/examples/python

[group('examples')]
examples: examples-go examples-nodejs examples-python

[group('examples')]
[group('test')]
test-example-up:
  #!/usr/bin/env sh
  set -eu
  cd examples/yaml
  export PULUMI_CONFIG_PASSPHRASE=asdfqwerty1234
  pulumi login --local
  pulumi stack init dev
  pulumi stack select dev
  pulumi config set name dev
  pulumi up -y

[group('examples')]
[group('test')]
test-example-down:
  #!/usr/bin/env sh
  set -eu
  cd examples/yaml
  export PULUMI_CONFIG_PASSPHRASE=asdfqwerty1234
  pulumi login --local
  pulumi stack select dev
  pulumi destroy -y
  pulumi stack rm dev -y

[group('provider')]
install-provider:
  cp ./bin/pulumi-resource-mid "$(go env GOPATH)/bin"

[group('sdk')]
install-nodejs-sdk: sdk-nodejs
  #!/usr/bin/env sh
  cd ./sdk/nodejs/bin
  npm unlink @sapslaj/pulumi-mid
  npm link
  echo 'npm: run "npm link @sapslaj/pulumi-mid" in a project to link to local build'
  if command -v yarn &>/dev/null; then
    yarn unlink
    yarn link
  fi

[group('tools')]
install-tools: install-tool-pulumi install-tool-dprint install-tool-gci install-tool-goimports

[group('tools')]
install-tool-pulumi:
  #!/usr/bin/env sh
  curl -fsSL https://get.pulumi.com | sh

[group('tools')]
install-tool-dprint:
  #!/usr/bin/env sh
  curl -fsSL https://dprint.dev/install.sh | sh

[group('tools')]
install-tool-gci:
  mkdir -p ./bin
  GOBIN=`pwd`/bin go install github.com/daixiang0/gci@v0.13.6

[group('tools')]
install-tool-goimports:
  mkdir -p ./bin
  GOBIN=`pwd`/bin go install golang.org/x/tools/cmd/goimports@latest
