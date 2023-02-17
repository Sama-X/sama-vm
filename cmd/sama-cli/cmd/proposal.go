// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
)

var proposalCmd = &cobra.Command{
	Use:   "proposal  [options] <param> <value>",
	Short: "proposal serPerc 30",
	RunE:  proposalFunc,
}

func proposalFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	actionType, key, newValue, err := getProposalOp(args)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)

	actionID, err := cli.CreateShortID(context.Background())
	if err != nil {
		return err
	}
	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	startTime := uint64(time.Now().Unix())
	endTime := startTime + chain.Seconds7Day

	utx := &chain.ProposalTx{
		BaseTx:     &chain.BaseTx{},
		ActionID:   actionID,
		StartTime:  startTime,
		EndTime:    endTime,
		Key:        key,
		NewValue:   newValue,
		ActionType: actionType,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("Proposal actionID=%s", actionID.String())
	return nil
}

func getProposalOp(args []string) (actionType uint64, key string, newValue string, err error) {
	if len(args) != 3 {
		return 0, "", "", fmt.Errorf("expected exactly 3 argument, got %d", len(args))
	}

	actionType, err = strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return 0, "", "", fmt.Errorf("%w: failed to parse actionType", err)
	}
	key = args[1]
	newValue = args[2]

	return actionType, key, newValue, nil
}
