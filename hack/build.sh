#!/usr/bin/env bash

# Copyright AppsCode Inc. and Contributors
#
# Licensed under the AppsCode Community License 1.0.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eou pipefail

if [ -z "${OS:-}" ]; then
    echo "OS must be set"
    exit 1
fi
if [ -z "${ARCH:-}" ]; then
    echo "ARCH must be set"
    exit 1
fi
if [ -z "${VERSION:-}" ]; then
    echo "VERSION must be set"
    exit 1
fi

if [ -z "${RESTIC_VER:-}" ]; then
    echo "RESTIC_VER must be set"
    exit 1
fi

export CGO_ENABLED=0
export GOARCH="${ARCH}"
export GOOS="${OS}"
export GO111MODULE=on
export GOFLAGS="-mod=vendor"

go install \
    -installsuffix "static" \
    -ldflags "                                          \
      -X main.Version=${VERSION}                        \
      -X main.VersionStrategy=${version_strategy:-}     \
      -X main.GitTag=${git_tag:-}                       \
      -X main.GitBranch=${git_branch:-}                 \
      -X main.CommitHash=${commit_hash:-}               \
      -X main.CommitTimestamp=${commit_timestamp:-}     \
      -X main.GoVersion=$(go version | cut -d " " -f 3) \
      -X main.Compiler=$(go env CC)                     \
      -X main.Platform=${OS}/${ARCH}                    \
      -X stash.appscode.dev/cli/pkg.ResticTag=${RESTIC_VER} \
    " \
    ./...
