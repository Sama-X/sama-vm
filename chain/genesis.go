// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/inconshreveable/log15"
)

const (
	MinBlockCost          = 0
	DefaultValueUnitSize  = 1 * units.KiB
	DefaultLookbackWindow = 60
)

type Airdrop struct {
	// Address strings are hex-formatted common.Address
	Address common.Address `serialize:"true" json:"address"`
}

type CustomAllocation struct {
	// Address strings are hex-formatted common.Address
	Address common.Address `serialize:"true" json:"address"`
	Balance uint64         `serialize:"true" json:"balance"`
}

type Genesis struct {
	Magic uint64 `serialize:"true" json:"magic"`

	// Tx params
	BaseTxUnits uint64 `serialize:"true" json:"baseTxUnits"`

	// SetTx params
	ValueUnitSize uint64 `serialize:"true" json:"valueUnitSize"`
	MaxValueSize  uint64 `serialize:"true" json:"maxValueSize"`

	// Fee Mechanism Params
	MinPrice         uint64 `serialize:"true" json:"minPrice"`
	LookbackWindow   int64  `serialize:"true" json:"lookbackWindow"`
	TargetBlockRate  int64  `serialize:"true" json:"targetBlockRate"` // seconds
	TargetBlockSize  uint64 `serialize:"true" json:"targetBlockSize"` // units
	MaxBlockSize     uint64 `serialize:"true" json:"maxBlockSize"`    // units
	BlockCostEnabled bool   `serialize:"true" json:"blockCostEnabled"`

	// Allocations
	CustomAllocation []*CustomAllocation `serialize:"true" json:"customAllocation"`
	AirdropHash      string              `serialize:"true" json:"airdropHash"`
	AirdropUnits     uint64              `serialize:"true" json:"airdropUnits"`

	//chain Params
	TotalTokens      uint64 `serialize:"true" json:"totalTokens"`
	ClaimMinUnits    uint64 `serialize:"true" json:"claimMinUnits"`
	ClaimMinInterval uint64 `serialize:"true" json:"claimMinInterval"`
	TotalYears       uint32 `serialize:"true" json:"totalYears"`
	RateSustainYears uint32 `serialize:"true" json:"rateChangeYears"`

	ChainCreateTime uint64 `serialize:"true" json:"chainCreateTime"`

	MinerPerc      uint32 `serialize:"true" json:"minerPerc"`
	FoundationPerc uint32 `serialize:"true" json:"foundationPerc"`

	RoutePerc     uint32 `serialize:"true" json:"routePerc"`
	SerPerc       uint32 `serialize:"true" json:"serPerc"`
	ValidatorPerc uint32 `serialize:"true" json:"validatorPerc"`

	BaseSerPerc    uint32 `serialize:"true" json:"baseSer"`
	MeritSerPerc   uint32 `serialize:"true" json:"meritSerPerc"`
	BaseRoutePerc  uint32 `serialize:"true" json:"baseRoutePerc"`
	MeritRoutePerc uint32 `serialize:"true" json:"meritRoutePerc"`

	BurnPerc uint32 `serialize:"true" json:"burnPerc"`

	RouteStake uint64 `serialize:"true" json:"routeAmount"`
	SerStake   uint64 `serialize:"true" json:"serAmount"`

	MonthCard  uint64 `serialize:"true" json:"month"`
	SeasonCard uint64 `serialize:"true" json:"season"`
	AnnualCard uint64 `serialize:"true" json:"annual"`

	MinStakeTime uint64 `serialize:"true" json:"minStakeTime"`

	RootAddress    string `serialize:"true" json:"rootAddress"`
	FoundationAddr string `serialize:"true" json:"foundation"`
}

func DefaultGenesis() *Genesis {
	return &Genesis{
		// Tx params
		BaseTxUnits: 1,

		// SetTx params
		ValueUnitSize: DefaultValueUnitSize,
		MaxValueSize:  200 * units.KiB,

		// Fee Mechanism Params
		LookbackWindow:   DefaultLookbackWindow, // 60 Seconds
		TargetBlockRate:  1,                     // 1 Block per Second
		TargetBlockSize:  225,                   // ~225KB
		MaxBlockSize:     246,                   // ~246KB -> Limited to 256KB by AvalancheGo (as of v1.7.3)
		MinPrice:         0,
		BlockCostEnabled: true,

		//chain Params
		TotalTokens:      2100000000000,
		ClaimMinUnits:    100,
		ChainCreateTime:  1668125288,
		ClaimMinInterval: 7 * 24 * 60 * 60,
		TotalYears:       10,
		RateSustainYears: 1,

		MinStakeTime: 90 * 24 * 60 * 60,

		MinerPerc:      20,
		FoundationPerc: 40,

		RoutePerc:     30,
		SerPerc:       50,
		ValidatorPerc: 20,

		BaseSerPerc:  20,
		MeritSerPerc: 80,

		BaseRoutePerc:  20,
		MeritRoutePerc: 80,

		RouteStake: 200000000,
		SerStake:   100000000,

		MonthCard:  100,
		SeasonCard: 300,
		AnnualCard: 1200,

		BurnPerc:       20,
		RootAddress:    "0x8db97c7cece249c2b98bdc0226cc4c2a57bf52fc",
		FoundationAddr: "0x8db97c7cece249c2b98bdc0226cc4c2a57bf52fc",
	}
}

func (g *Genesis) StatefulBlock() *StatefulBlock {
	return &StatefulBlock{
		Price: g.MinPrice,
		Cost:  MinBlockCost,
	}
}

func (g *Genesis) Verify() error {
	if g.Magic == 0 {
		return ErrInvalidMagic
	}
	if g.TargetBlockRate == 0 {
		return ErrInvalidBlockRate
	}
	return nil
}

func (g *Genesis) Load(db database.Database, airdropData []byte) error {
	start := time.Now()
	defer func() {
		log.Debug("loaded genesis allocations", "t", time.Since(start))
	}()

	vdb := versiondb.New(db)
	if len(g.AirdropHash) > 0 {
		h := common.BytesToHash(crypto.Keccak256(airdropData)).Hex()
		if g.AirdropHash != h {
			return fmt.Errorf("expected standard allocation %s but got %s", g.AirdropHash, h)
		}

		airdrop := []*Airdrop{}
		if err := json.Unmarshal(airdropData, &airdrop); err != nil {
			return err
		}

		for _, alloc := range airdrop {
			if err := SetBalance(vdb, alloc.Address, g.AirdropUnits); err != nil {
				return fmt.Errorf("%w: addr=%s, bal=%d", err, alloc.Address, g.AirdropUnits)
			}
		}
		log.Debug(
			"applied airdrop allocation",
			"hash", h, "addrs", len(airdrop), "balance", g.AirdropUnits,
		)
	}

	// Do custom allocation last in case an address shows up in standard
	// allocation
	for _, alloc := range g.CustomAllocation {
		if err := SetBalance(vdb, alloc.Address, alloc.Balance); err != nil {
			return fmt.Errorf("%w: addr=%s, bal=%d", err, alloc.Address, alloc.Balance)
		}
		log.Debug("applied custom allocation", "addr", alloc.Address, "balance", alloc.Balance)
	}

	// Commit as a batch to improve speed
	return vdb.Commit()
}
