// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "sama-cli" implements samavm client operation interface.
package main

import (
	"os"

	"github.com/fatih/color"

	"github.com/SamaNetwork/SamaVM/cmd/sama-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		color.Red("sama-cli failed: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}
