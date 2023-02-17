// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/SamaNetwork/SamaVM/client"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config [options] <identity>",
	Short: "Configs a new key in the default location",
	Long: `
Configs a new key in the default location.
It will error if the key file already exists.
need config before register route or ser node

$ sama-cli config

`,
	RunE: configFunc,
}

func configFunc(_ *cobra.Command, args []string) error {
	filePath := "./sama-work-pk"
	privateKey, userName, userPass, err := getConfigOp(args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filePath); err == nil {
		// Already found, remind the user they have it
		priv, err := crypto.LoadECDSA(filePath)
		if err != nil {
			return err
		}
		color.Green("ABORTING!!! key for %s already exists at %s", crypto.PubkeyToAddress(priv.PublicKey), filePath)
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := "username:"
	text += userName
	text += "\npassword:"
	text += userPass
	text += "\nprivateKey:"
	if privateKey == "" {
		// Generate new key and save to disk
		// TODO: encrypt key
		priv, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		if err := crypto.SaveECDSA("work_pk", priv); err != nil {
			return err
		}

		privateKey = hex.EncodeToString(crypto.FromECDSA(priv))
		text += privateKey

		text += "\npublicKey:"
		pbk := hex.EncodeToString(crypto.FromECDSAPub(&priv.PublicKey))
		text += pbk
		text += "\naddress:"
		text += crypto.PubkeyToAddress(priv.PublicKey).String()

		color.Green("configd %s address %s and saved to %s", pbk, crypto.PubkeyToAddress(priv.PublicKey), filePath)
	} else {
		text += privateKey
		priv, err := crypto.HexToECDSA(privateKey)
		if err != nil {
			return err
		}
		if err := crypto.SaveECDSA("work_pk", priv); err != nil {
			return err
		}
		privateKey = string(crypto.FromECDSA(priv))

		pbk := hex.EncodeToString(crypto.FromECDSAPub(&priv.PublicKey))
		text += "\npublicKey:"
		text += pbk
		text += "\naddress:"
		text += crypto.PubkeyToAddress(priv.PublicKey).String()

		color.Green("configd %s address %s and saved to %s", pbk, crypto.PubkeyToAddress(priv.PublicKey), filePath)
	}
	text += "\n"
	err = os.WriteFile(filePath, []byte(text), 0600)
	if err != nil {
		return err
	}

	cli := client.New(uri, requestTimeout)
	err = cli.ImportKey(context.Background(), userName, userPass, privateKey)
	if err != nil {
		return err
	}
	return nil
}

func getConfigOp(args []string) (privateKey string, userName string, userPass string, err error) {
	if len(args) != 0 && len(args) != 1 && len(args) != 2 && len(args) != 3 {
		return "", "", "", fmt.Errorf("expected exactly 0 1 2 3 argument, got %d", len(args))
	}
	privateKey = ""
	userName = "samawork"
	userPass = "sama64854331"
	if len(args) == 1 {
		privateKey = args[0]
	} else if len(args) == 2 {
		userName = args[0]
		userPass = args[1]
	} else if len(args) == 3 {
		privateKey = args[0]
		userName = args[1]
		userPass = args[2]
	} else {
		return "", "", "", nil
	}
	return privateKey, userName, userPass, nil
}
