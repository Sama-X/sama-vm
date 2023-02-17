// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package version implements "version" commands.
package version

import (
	"fmt"

	"github.com/SamaNetwork/SamaVM/version"
	"github.com/SamaNetwork/SamaVM/vm"
	"github.com/spf13/cobra"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "samavm version" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints out the verson",
		RunE:  versionFunc,
	}
	return cmd
}

func versionFunc(_ *cobra.Command, _ []string) error {
	fmt.Printf("%s@%s\n", vm.Name, version.Version)
	return nil
}
