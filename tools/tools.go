//go:build tools
// +build tools

// This package contains import references to packages required only for the
// build process.
//
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
