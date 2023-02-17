// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
	"github.com/ava-labs/avalanchego/ids"
)

var voteCmd = &cobra.Command{
	Use:   "vote  [options]  <ActionID> ",
	Short: "vote for action",
	RunE:  voteFunc,
}

func voteFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	actionID, err := getVoteOp(args)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	utx := &chain.VoteTx{
		BaseTx:   &chain.BaseTx{},
		ActionID: actionID,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("Vote actionID=%s", actionID.String())
	return nil
}

func getVoteOp(args []string) (actionID ids.ShortID, err error) {
	if len(args) != 1 {
		return ids.ShortID{}, fmt.Errorf("expected exactly 1 argument, got %d", len(args))
	}

	actionID, _ = ids.ShortFromString(args[0])

	return actionID, nil
}
