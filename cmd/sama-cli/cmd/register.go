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

var zeroAddress = (common.Address{})

var registerCmd = &cobra.Command{
	Use:   "register  [options]  <StakerType> ",
	Short: "addr register in order to ser node or route node",
	RunE:  registerFunc,
}

func registerFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	stakerType, stakerAddr, wrokAddr, err := getRegisterOp(args)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)

	actionID, err := cli.CreateShortID(context.Background())
	if err != nil {
		return err
	}

	params, err := cli.GetLocalParams(context.Background(), wrokAddr)
	if err != nil {
		return fmt.Errorf("need start work module %w", err)
	}

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	if stakerAddr == zeroAddress {
		stakerAddr = crypto.PubkeyToAddress(priv.PublicKey)
	}

	utx := &chain.RegisterTx{
		BaseTx:     &chain.BaseTx{},
		StakerType: stakerType,
		ActionID:   actionID,
		StakerAddr: stakerAddr,
		Country:    params.Country,
		WorkKey:    params.WorkKey,
		LocalIP:    params.LocalIP,
		MinPort:    params.MinPort,
		MaxPort:    params.MaxPort,
		PublicIP:   params.PublicIP,
		CheckPort:  params.CheckPort,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("%s Register %d ActionID=%s", stakerAddr.String(), stakerType, actionID.String())
	return nil
}

func getRegisterOp(args []string) (stakerType uint64, stakeAddress common.Address, workAddress common.Address, err error) {
	if len(args) != 2 && len(args) != 3 {
		return 0, common.Address{}, common.Address{}, fmt.Errorf("expected exactly 2-3argument, got %d", len(args))
	}

	stakerType, err = strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return 0, common.Address{}, common.Address{}, fmt.Errorf("%w: failed to parse stakerType", err)
	}

	workAddress = common.HexToAddress(args[1])

	if len(args) == 3 {
		stakeAddress = common.HexToAddress(args[2])
		return stakerType, stakeAddress, workAddress, nil
	}
	return stakerType, common.Address{}, workAddress, nil

}
