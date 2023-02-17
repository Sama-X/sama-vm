#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/tests.unit.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

SamaVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SamaVM_PATH"/scripts/constants.sh

go test -race -timeout="3m" -coverprofile="coverage.out" -covermode="atomic" $(go list ./... | grep -v tests)
