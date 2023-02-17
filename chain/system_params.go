// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	ParamPercRoute = 1
	ParamPercSer   = 2
	ParamPercBase  = 3
	ParamPercMerit = 4
	ParamPercBurn  = 5

	MinPercentage = 5
	MaxPercentage = 95
)

func PrefixSysParamsKey() (k []byte) {
	k = make([]byte, 2)
	k[0] = sysParamPrefix
	k[1] = ByteDelimiter
	return
}

var _ SysParams = &sysParams{}

type SysParamsMeta struct {
	Symbol           string `serialize:"true" json:"symbol"`
	TotalTokens      uint64 `serialize:"true" json:"totalTokens"`
	ClaimMinUnits    uint64 `serialize:"true" json:"claimMinUnits"`
	ClaimMinInterval uint64 `serialize:"true" json:"claimMinInterval"`
	TotalYears       uint32 `serialize:"true" json:"totalYears"`
	RateSustainYears uint32 `serialize:"true" json:"rateChangeYears"`
	ChainCreateTime  uint64 `serialize:"true" json:"chainCreateTime"`
	MinerPerc        uint32 `serialize:"true" json:"minerPerc"`
	FoundationPerc   uint32 `serialize:"true" json:"foundationPerc"`
	RoutePerc        uint32 `serialize:"true" json:"routePerc"`
	SerPerc          uint32 `serialize:"true" json:"serPerc"`
	ValidatorPerc    uint32 `serialize:"true" json:"validatorPerc"`
	BaseSerPerc      uint32 `serialize:"true" json:"baseSer"`
	MeritSerPerc     uint32 `serialize:"true" json:"meritSerPerc"`
	BaseRoutePerc    uint32 `serialize:"true" json:"baseRoutePerc"`
	MeritRoutePerc   uint32 `serialize:"true" json:"meritRoutePerc"`
	RouteStakeAmount uint64 `serialize:"true" json:"routeAmount"`
	SerStakeAmount   uint64 `serialize:"true" json:"serAmount"`
	MinStakeTime     uint64 `serialize:"true" json:"minStakeTime"`
	MonthCard        uint64 `serialize:"true" json:"month"`
	SeasonCard       uint64 `serialize:"true" json:"season"`
	AnnualCard       uint64 `serialize:"true" json:"annual"`
	BurnPerc         uint32 `serialize:"true" json:"burnPerc"`
	RootAddress      string `serialize:"true" json:"rootAddress"`
	FoundationAddr   string `serialize:"true" json:"foundation"`
	UpdateTime       uint64 `serialize:"true" json:"updateTime"`
	UpdateTxID       ids.ID `serialize:"true" json:"updateTxId"`
}

type SysParams interface {
	ReloadParams(db database.Database) error
	ModifyParams(db database.Database, key string, newValue string, txID ids.ID, updateTime uint64) error
	CacheParamsCommit() error
	CacheParamsAbort() error
	GetChainSymbol() string
	GetChainTokenAmount() uint64
	GetPercRoute() uint32
	GetPercSer() uint32
	GetPercValidator() uint32
	GetRoutePercBase() uint32
	GetRoutePercMerit() uint32
	GetSerPercBase() uint32
	GetSerPercMerit() uint32
	GetPercBurn() uint32
	GetPercFoundation() uint32
	GetTotalYears() uint32
	GetRateSustainYears() uint32
	GetChainCreateTime() uint64
	GetMonthCardPrice() uint64
	GetSeasonCardPrice() uint64
	GetAnnualCardPrice() uint64
	GetMinStakeTime() uint64
	GetRootAddress() string
	GetSysParams() *SysParamsMeta
	CompCurParam(key string, newValue string) error
	GetFoundationAddress() string
}

type sysParams struct {
	curParams     *SysParamsMeta
	pendingParams *SysParamsMeta
}

func NewSysParamsState(db database.Database, genesis *Genesis) (*sysParams, error) {
	curParams := &SysParamsMeta{
		TotalTokens:      genesis.TotalTokens,
		ClaimMinUnits:    genesis.ClaimMinUnits,
		ChainCreateTime:  genesis.ChainCreateTime,
		ClaimMinInterval: genesis.ClaimMinInterval,
		TotalYears:       genesis.TotalYears,
		RateSustainYears: genesis.RateSustainYears,
		RoutePerc:        genesis.RoutePerc,
		SerPerc:          genesis.SerPerc,
		MinerPerc:        genesis.MinerPerc,
		BaseSerPerc:      genesis.BaseSerPerc,
		MeritSerPerc:     genesis.MeritSerPerc,
		BaseRoutePerc:    genesis.BaseRoutePerc,
		MeritRoutePerc:   genesis.MeritRoutePerc,

		RouteStakeAmount: genesis.RouteStake,
		SerStakeAmount:   genesis.SerStake,
		RootAddress:      genesis.RootAddress,
		BurnPerc:         genesis.BurnPerc,
		MonthCard:        genesis.MonthCard,
		SeasonCard:       genesis.SeasonCard,
		AnnualCard:       genesis.AnnualCard,
		MinStakeTime:     genesis.MinStakeTime,
	}
	pendingParams := &SysParamsMeta{
		TotalTokens: 0,
	}
	state := &sysParams{
		curParams:     curParams,
		pendingParams: pendingParams,
	}
	err := state.ReloadParams(db)
	if err != nil {
		if err != database.ErrNotFound {
			return nil, err
		}
	}
	return state, nil
}

func (s *sysParams) GetChainSymbol() string {
	return s.curParams.Symbol
}

func (s *sysParams) GetChainTokenAmount() uint64 {
	return (s.curParams.TotalTokens / 100) * uint64(s.curParams.MinerPerc)
}

func (s *sysParams) GetPercRoute() uint32 {
	return s.curParams.RoutePerc
}

func (s *sysParams) GetPercSer() uint32 {
	return s.curParams.SerPerc
}

func (s *sysParams) GetPercValidator() uint32 {
	return s.curParams.ValidatorPerc
}

func (s *sysParams) GetRoutePercBase() uint32 {
	return s.curParams.BaseRoutePerc
}

func (s *sysParams) GetRoutePercMerit() uint32 {
	return s.curParams.MeritRoutePerc
}

func (s *sysParams) GetSerPercBase() uint32 {
	return s.curParams.BaseSerPerc
}

func (s *sysParams) GetSerPercMerit() uint32 {
	return s.curParams.MeritSerPerc
}

func (s *sysParams) GetPercBurn() uint32 {
	return s.curParams.BurnPerc
}

func (s *sysParams) GetPercFoundation() uint32 {
	return s.curParams.FoundationPerc
}

func (s *sysParams) GetTotalYears() uint32 {
	return s.curParams.TotalYears
}

func (s *sysParams) GetRateSustainYears() uint32 {
	return s.curParams.RateSustainYears
}

func (s *sysParams) GetChainCreateTime() uint64 {
	return s.curParams.ChainCreateTime
}

func (s *sysParams) GetMonthCardPrice() uint64 {
	return s.curParams.MonthCard
}

func (s *sysParams) GetSeasonCardPrice() uint64 {
	return s.curParams.SeasonCard
}

func (s *sysParams) GetAnnualCardPrice() uint64 {
	return s.curParams.AnnualCard
}

func (s *sysParams) GetRootAddress() string {
	return s.curParams.RootAddress
}

func (s *sysParams) GetFoundationAddress() string {
	return s.curParams.FoundationAddr
}

func (s *sysParams) GetMinStakeTime() uint64 {
	return s.curParams.MinStakeTime
}

func (s *sysParams) GetSysParams() *SysParamsMeta {
	return s.curParams
}

func (s *sysParams) ModifyParams(db database.Database, key string, newValue string, txID ids.ID, updateTime uint64) error {
	ymeta := new(SysParamsMeta)
	*ymeta = *(s.curParams)

	perc, err := strconv.ParseUint(newValue, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: failed to parse newValue %s", err, newValue)
	}
	newPerc := uint32(perc)
	if newPerc > MaxPercentage || newPerc < MinPercentage {
		return fmt.Errorf("percentage err")
	}
	ymeta.UpdateTxID = txID
	ymeta.UpdateTime = updateTime

	if key == "routePerc" {
		ymeta.RoutePerc = newPerc
		ymeta.SerPerc = 100 - newPerc
	} else if key == "serPerc" {
		ymeta.SerPerc = newPerc
		ymeta.RoutePerc = 100 - newPerc
	} else if key == "routeBase" {
		ymeta.BaseRoutePerc = newPerc
		ymeta.MeritRoutePerc = 100 - newPerc
	} else if key == "routeMerit" {
		ymeta.MeritRoutePerc = newPerc
		ymeta.BaseRoutePerc = 100 - newPerc
	} else if key == "serBase" {
		ymeta.BaseSerPerc = newPerc
		ymeta.MeritSerPerc = 100 - newPerc
	} else if key == "serMerit" {
		ymeta.MeritSerPerc = newPerc
		ymeta.BaseSerPerc = 100 - newPerc
	} else if key == "burn" {
		ymeta.BurnPerc = newPerc
	} else {
		return fmt.Errorf("err key %s", key)
	}
	*s.pendingParams = *ymeta

	k := PrefixSysParamsKey()
	pvmeta, err := Marshal(ymeta)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (s *sysParams) ReloadParams(db database.Database) error {
	k := PrefixSysParamsKey()
	ymeta, err := db.Get(k)
	if err != nil {
		return nil
	}
	if _, err := Unmarshal(ymeta, s.curParams); err != nil {
		return err
	}
	return nil
}

func (s *sysParams) CacheParamsCommit() error {
	if s.pendingParams.TotalTokens != 0 {
		*s.curParams = *s.pendingParams
		s.pendingParams.TotalTokens = 0
	}
	return nil
}

func (s *sysParams) CacheParamsAbort() error {
	s.pendingParams.TotalTokens = 0
	return nil
}

func (s *sysParams) CompCurParam(key string, newValue string) error {
	oldPerc := uint32(0)
	newPerc, err := strconv.ParseUint(newValue, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: failed to parse newValue %s", err, newValue)
	}
	if key == "routePerc" {
		oldPerc = s.curParams.RoutePerc
	} else if key == "serPerc" {
		oldPerc = s.curParams.SerPerc
	} else if key == "routeBase" {
		oldPerc = s.curParams.BaseRoutePerc
	} else if key == "routeMerit" {
		oldPerc = s.curParams.MeritRoutePerc
	} else if key == "serBase" {
		oldPerc = s.curParams.BaseSerPerc
	} else if key == "serMerit" {
		oldPerc = s.curParams.MeritSerPerc
	} else if key == "burn" {
		oldPerc = s.curParams.BurnPerc
	} else {
		return fmt.Errorf("err key %s", key)
	}
	if uint64(oldPerc) == newPerc {
		return fmt.Errorf("equal CurParam")
	}
	return nil
}
