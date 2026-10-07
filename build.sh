#!/usr/bin/env bash
set -e
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
go build -tags netgo -ldflags "-s -w -X main.Version=${VERSION}" -o app .
