// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
)

var stakeCmd = &cobra.Command{
	Use:   "stake  [options]  <stakerType> <stakeAmount> <stakeAddress>",
	Short: "addr stake xxx token in order to ser node or route node",
	RunE:  stakeFunc,
}

func stakeFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	stakerType, stakeAmount, stakerAddr, err := getStakeOp(args)
	if err != nil {
		return err
	}
	utx := &chain.StakeTx{
		BaseTx:      &chain.BaseTx{},
		StakerType:  stakerType,
		StakeAmount: stakeAmount,
		StakerAddr:  stakerAddr,
	}
	cli := client.New(uri, requestTimeout)
	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("%s staker %d", stakerAddr.Hex(), stakeAmount)
	return nil
}

func getStakeOp(args []string) (stakerType uint64, stakeAount uint64, stakerAddr common.Address, err error) {
	if len(args) != 3 {
		return 0, 0, common.Address{}, fmt.Errorf("expected exactly 3 argument, got %d", len(args))
	}

	stakerType, _ = strconv.ParseUint(args[0], 10, 64)

	stakeAount, err = strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return 0, 0, common.Address{}, fmt.Errorf("%w: failed to parse stakeAount", err)
	}
	stakerAddr = common.HexToAddress(args[2])

	return stakerType, stakeAount, stakerAddr, nil
}
