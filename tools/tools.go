// Forked from https://github.com/moby/buildkit/blob/e1b3b6c4abf7684f13e6391e5f7bc9210752687a/tools/tools.go
//go:build tools
// +build tools

// Package tools tracks dependencies on binaries not referenced in this codebase.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// Disclaimer: Avoid adding tools that don't need to be inferred from go.mod
// like golangci-lint and check they don't import too many dependencies.
package tools

import (
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
