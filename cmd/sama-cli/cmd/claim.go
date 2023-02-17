// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/client"
)

var claimCmd = &cobra.Command{
	Use:   "claim  [options] ",
	Short: "claim reward",
	RunE:  claimFunc,
}

func claimFunc(_ *cobra.Command, _ []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}
	address := crypto.PubkeyToAddress(priv.PublicKey)

	cli := client.New(uri, requestTimeout)

	stakerType, err := cli.GetStakerType(context.Background(), address)
	if err != nil {
		return err
	}
	createTime, err := cli.GetChainCreateTime(context.Background())
	if err != nil {
		return err
	}
	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}
	color.Green("Claim %s ", address)
	endTime := ((uint64(time.Now().Unix())-createTime)/chain.SecondsDay)*chain.SecondsDay + createTime
	color.Green("Claim %d %d ", uint64(time.Now().Unix()), createTime)
	base, merit, yield, err := cli.CalcReward(context.Background(), stakerType, endTime, address)
	if err != nil {
		return err
	}
	rewardAmount := base + merit + yield
	color.Green("Claim %d %d %d ", base, merit, yield)
	utx := &chain.ClaimTx{
		BaseTx:       &chain.BaseTx{},
		RewardAmount: rewardAmount,
		EndTime:      endTime,
	}
	color.Green("Claim %d %s ", endTime, address)
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}
	color.Green("Claim base:%d merit:%d yield:%d endTime=%d", base, merit, yield, endTime)
	return nil
}
