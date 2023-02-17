// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "sama-cli" implements samavm client operation interface.
package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	requestTimeout = 30 * time.Second
	fsModeWrite    = 0o600
)

var (
	privateKeyFile string
	uri            string
	verbose        bool
	workDir        string

	rootCmd = &cobra.Command{
		Use:        "sama-cli",
		Short:      "SamaVM CLI",
		SuggestFor: []string{"sama-cli", "samacli"},
	}
)

func init() {
	p, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	workDir = p
	cobra.EnablePrefixMatching = true
	rootCmd.AddCommand(
		createCmd,
		genesisCmd,
		setCmd,
		resolveCmd,
		activityCmd,
		transferCmd,
		setFileCmd,
		resolveFileCmd,
		networkCmd,
		stakeCmd,
		unStakeCmd,
		registerCmd,
		voteCmd,
		addUserCmd,
		claimCmd,
		proofCmd,
		refreshCmd,
		proposalCmd,
		governCmd,
		configCmd,
	)

	rootCmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		"./sama-cli-pk",
		"private key file path",
	)
	rootCmd.PersistentFlags().StringVar(
		&uri,
		"endpoint",
		"http://127.0.0.1:9650/ext/bc/2i8WsFjx9ACX1PLzg33KGi9s4k8YAUZfy3b6jw1Xs9e9asB52n",
		"RPC endpoint for VM",
	)
	rootCmd.PersistentFlags().BoolVar(
		&verbose,
		"verbose",
		false,
		"Print verbose information about operations",
	)
}

func Execute() error {
	return rootCmd.Execute()
}
