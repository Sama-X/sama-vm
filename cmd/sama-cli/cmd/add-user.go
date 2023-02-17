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

var addUserCmd = &cobra.Command{
	Use:   "addUser  [options]  <address> <type>",
	Short: "add User to sama ",
	RunE:  addUserFunc,
}

func addUserFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}

	userType, startTime, endTime, address, err := getAddUserOp(args)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	payAmount, err := cli.GetUserFee(context.Background(), userType, startTime, endTime)
	if err != nil {
		return err
	}
	utx := &chain.AddUserTx{
		BaseTx:    &chain.BaseTx{},
		StartTime: startTime,
		EndTime:   endTime,
		PayAmount: payAmount,
		Address:   address,
		UserType:  userType,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("add user %s startTime=%d, endTime=%d fee=%d", address.String(), startTime, endTime, payAmount)
	return nil
}

func getAddUserOp(args []string) (userType uint64, startTime uint64, endTime uint64, address common.Address, err error) {
	if len(args) != 2 {
		return 0, 0, 0, common.Address{}, fmt.Errorf("expected exactly 2 argument, got %d", len(args))
	}
	address = common.HexToAddress(args[0])

	userType, err = strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return 0, 0, 0, common.Address{}, fmt.Errorf("%w: failed to parse userType", err)
	}

	startTime = uint64(time.Now().Unix())

	endTime = startTime + chain.SecondsMonth

	return userType, startTime, endTime, address, nil
}
