// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
)

var governCmd = &cobra.Command{
	Use:   "govern  [options]  <params> ",
	Short: "govern node params",
	RunE:  governFunc,
}

func governFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}
	//address := crypto.PubkeyToAddress(priv.PublicKey)
	actionID, err := getGovernOp(args)
	if err != nil {
		return err
	}
	cli := client.New(uri, requestTimeout)

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	utx := &chain.GovernTx{
		BaseTx:   &chain.BaseTx{},
		ActionID: actionID,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("Govern actionID=%s", actionID.Hex())
	return nil
}

func getGovernOp(args []string) (actionID ids.ShortID, err error) {
	if len(args) != 1 {
		return ids.ShortID{}, fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}

	actionID, _ = ids.ShortFromString(args[0])

	return actionID, nil
}
