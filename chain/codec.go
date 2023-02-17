// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

const (
	// codecVersion is the current default codec version
	codecVersion = 0

	// maxSize is 4MB to support large values
	maxSize = 4 * units.MiB
)

var codecManager codec.Manager

func init() {
	c := linearcodec.NewDefault()
	codecManager = codec.NewManager(maxSize)
	errs := wrappers.Errs{}
	errs.Add(
		c.RegisterType(&BaseTx{}),
		c.RegisterType(&SetTx{}),
		c.RegisterType(&TransferTx{}),
		c.RegisterType(&Transaction{}),
		c.RegisterType(&StatefulBlock{}),
		c.RegisterType(&CustomAllocation{}),
		c.RegisterType(&Airdrop{}),
		c.RegisterType(&Genesis{}),
		c.RegisterType(&StakeTx{}),
		c.RegisterType(&UnStakeTx{}),

		c.RegisterType(&ClaimTx{}),
		c.RegisterType(&RegisterTx{}),
		c.RegisterType(&RefreshTx{}),
		c.RegisterType(&AddUserTx{}),
		c.RegisterType(&GovernTx{}),
		c.RegisterType(&VoteTx{}),
		c.RegisterType(&ProofTx{}),
		c.RegisterType(&ProposalTx{}),

		codecManager.RegisterCodec(codecVersion, c),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}

func Marshal(source interface{}) ([]byte, error) {
	return codecManager.Marshal(codecVersion, source)
}

func Unmarshal(source []byte, destination interface{}) (uint16, error) {
	return codecManager.Unmarshal(source, destination)
}
