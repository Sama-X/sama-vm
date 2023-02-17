// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/SamaNetwork/SamaVM/tdata"
)

var _ UnsignedTransaction = &UnStakeTx{}
var EffectiveSecs = uint64(100)

type UnStakeTx struct {
	*BaseTx      `serialize:"true" json:"baseTx"`
	RewardAmount uint64 `serialize:"true" json:"rewardAmount"`
	StakerType   uint64 `serialize:"true" json:"stakerType"`
	EndTime      uint64 `serialize:"true" json:"endTime"`
}

func (u *UnStakeTx) Execute(t *TransactionContext) error {
	switch {
	case u.StakerType != stakerTypeRoute && u.StakerType != stakerTypeSer:
		return ErrStakerType
	}

	samaState := t.vm.SamaState()

	if time.Now().After(time.Unix(int64(u.EndTime+EffectiveSecs), 0)) {
		return ErrEndTimeTooEarly
	}

	if time.Now().Before(time.Unix(int64(u.EndTime), 0)) {
		return ErrEndTimeTooLate
	}

	staker, exists, err := samaState.GetStakerMeta(byte(u.StakerType), t.Sender)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("not found")
	}

	minTime := samaState.GetMinStakeTime()

	if time.Now().Before(time.Unix(int64(staker.StakeTime+minTime), 0)) {
		return fmt.Errorf("stake time must > 90 days")
	}

	if _, err := ModifyStakeBalance(t.Database, t.Sender, false, staker.StakeAmount); err != nil {
		return err
	}

	base, merit, yield, err := samaState.CalcReward(byte(u.StakerType), t.Sender, u.EndTime)
	if err != nil {
		return err
	}
	totalReawrd := base + merit + yield
	if u.RewardAmount != totalReawrd {
		return fmt.Errorf("reward amount is err")
	}

	if _, err := ModifyBalance(t.Database, t.Sender, true, staker.StakeAmount+uint64(totalReawrd)); err != nil {
		return err
	}

	return samaState.DealUnStakeTx(t.Database, byte(u.StakerType), t.Sender, t.TxID, u.EndTime)
}

func (u *UnStakeTx) FeeUnits(g *Genesis) uint64 {
	return u.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(u.Value)))
}

func (u *UnStakeTx) LoadUnits(g *Genesis) uint64 {
	return u.FeeUnits(g)
}

func (u *UnStakeTx) Copy() UnsignedTransaction {
	return &UnStakeTx{
		BaseTx:     u.BaseTx.Copy(),
		StakerType: u.StakerType,
	}
}

func (u *UnStakeTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		u.Magic, UnStake,
		[]tdata.Type{
			{Name: tdStakerType, Type: tdUint64},
			{Name: tdAmount, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdStakerType: strconv.FormatUint(u.StakerType, 10),
			tdAmount:     strconv.FormatUint(u.RewardAmount, 10),
			tdPrice:      strconv.FormatUint(u.Price, 10),
			tdBlockID:    u.BlockID.String(),
		},
	)
}

func (u *UnStakeTx) Activity() *Activity {
	return &Activity{
		Typ:          UnStake,
		RewardAmount: u.RewardAmount,
		StakerType:   u.StakerType,
	}
}
