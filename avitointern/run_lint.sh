#!/bin/bash

set -exuo pipefail

golangci-lint -c .golangci.yml run ./...
