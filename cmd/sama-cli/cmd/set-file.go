// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/client"
	"github.com/SamaNetwork/SamaVM/tree"
)

var setFileCmd = &cobra.Command{
	Use:   "set-file [options] <file path>",
	Short: "Writes a file to SamaVM (using multiple keys)",
	RunE:  setFileFunc,
}

func setFileFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	f, err := getSetFileOp(args)
	if err != nil {
		return err
	}
	defer f.Close()

	cli := client.New(uri, requestTimeout)
	g, err := cli.Genesis(context.Background())
	if err != nil {
		return err
	}

	// TODO: protect against overflow
	root, err := tree.Upload(context.Background(), cli, priv, f, int(g.MaxValueSize))
	if err != nil {
		return err
	}

	color.Green("uploaded file %v from %s", root, f.Name())
	return nil
}

func getSetFileOp(args []string) (f *os.File, err error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("expected exactly 1 arguments, got %d", len(args))
	}

	filePath := args[0]
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("%w: file is not accessible", err)
	}

	f, err = os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open file", err)
	}

	return f, nil
}
