#!/bin/bash
set -euo pipefail
version="${1?}"
cd "$(dirname "${BASH_SOURCE[0]}")"/..
sed -i 's/Version = ".*"/Version = "'"${version}"'"/' ./version/version.go
