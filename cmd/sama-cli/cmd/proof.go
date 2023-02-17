// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
)

var proofCmd = &cobra.Command{
	Use:   "proof  [options]  <netflow> <worktime> ",
	Short: "work proof",
	RunE:  proofFunc,
}

func proofFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	address := crypto.PubkeyToAddress(priv.PublicKey)

	netflow, startTime, endTime, err := getProofOp(args)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	utx := &chain.ProofTx{
		BaseTx:    &chain.BaseTx{},
		Netflow:   netflow,
		StartTime: startTime,
		EndTime:   endTime,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("Proof %s netflow:%d, startTime=%d endTime=%d", address.String(), netflow, startTime, endTime)
	return nil
}

func getProofOp(args []string) (netflow uint64, startTime uint64, endTime uint64, err error) {
	if len(args) != 3 {
		return 0, 0, 0, fmt.Errorf("expected exactly 3 argument, got %d", len(args))
	}

	netflow, err = strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: failed to parse netflow", err)
	}

	startTime, err = strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: failed to parse startTime", err)
	}

	endTime, err = strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: failed to parse endTime", err)
	}

	return netflow, startTime, endTime, nil
}
