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

var refreshCmd = &cobra.Command{
	Use:   "refresh  [options]  <params> ",
	Short: "refresh node params",
	RunE:  refreshFunc,
}

func refreshFunc(_ *cobra.Command, args []string) error {
	priv, err := crypto.LoadECDSA(privateKeyFile)
	if err != nil {
		return err
	}
	address := crypto.PubkeyToAddress(priv.PublicKey)

	cli := client.New(uri, requestTimeout)

	node, err := cli.GetNodes(context.Background(), address)
	if err != nil {
		return err
	}
	localIP := node.LocalIP
	minPort := node.MinPort
	maxPort := node.MaxPort
	publicIP := node.PublicIP
	checkPort := node.CheckPort
	country := node.Country
	workKey := node.WorkKey

	key, value, err := getRefreshOp(args)
	if err != nil {
		return err
	}
	if key == "localIP" {
		localIP = value
	} else if key == "minPort" {
		minPort, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: failed to parse minPort", err)
		}
	} else if key == "maxPort" {
		maxPort, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: failed to parse maxPort", err)
		}
	} else if key == "publicIP" {
		publicIP = value
	} else if key == "checkPort" {
		checkPort, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: failed to parse checkPort", err)
		}
	} else if key == "country" {
		country = value
	} else {
		return fmt.Errorf("err key %s", key)
	}

	opts := []client.OpOption{client.WithPollTx()}
	if verbose {
		opts = append(opts, client.WithBalance())
	}

	utx := &chain.RefreshTx{
		BaseTx:    &chain.BaseTx{},
		Country:   country,
		WorkKey:   workKey,
		LocalIP:   localIP,
		MinPort:   minPort,
		MaxPort:   maxPort,
		PublicIP:  publicIP,
		CheckPort: checkPort,
	}
	if _, _, err := client.SignIssueRawTx(context.Background(), cli, utx, priv, opts...); err != nil {
		return err
	}

	color.Green("Refresh %s=%s", key, value)
	return nil
}

func getRefreshOp(args []string) (key string, value string, err error) {
	if len(args) != 2 {
		return "", "", fmt.Errorf("expected exactly 2 argument, got %d", len(args))
	}
	return args[0], args[1], nil
}
