// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
)

var unStakeCmd = &cobra.Command{
	Use:   "unstake  [options]  <StakerType> <unStakeAmount> <unStakeAddress>",
	Short: "addr unstake xxx token in order to ser node or route node",
	RunE:  unStakeFunc,
}

func unStakeFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	stakerType, address, err := getUnStakeOp(args)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)

	endTime := uint64(time.Now().Unix())

	base, merit, yield, err := cli.CalcReward(context.Background(), stakerType, endTime, address)
	if err != nil {
		return err
	}
	rewardAmount := base + merit + yield
	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	utx := &chain.UnStakeTx{
		BaseTx:       &chain.BaseTx{},
		StakerType:   stakerType,
		RewardAmount: rewardAmount,
		EndTime:      endTime,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("unstake %d", rewardAmount)
	return nil
}

func getUnStakeOp(args []string) (stakerType uint64, sender common.Address, err error) {
	if len(args) != 2 {
		return 0, common.Address{}, fmt.Errorf("expected exactly 2 argument, got %d", len(args))
	}

	stakerType, err = strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return 0, common.Address{}, fmt.Errorf("%w: failed to parse stakerType", err)
	}
	sender = common.HexToAddress(args[1])

	return stakerType, sender, nil
}
