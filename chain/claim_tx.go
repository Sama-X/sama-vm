// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/SamaNetwork/SamaVM/tdata"
)

var _ UnsignedTransaction = &ClaimTx{}

type ClaimTx struct {
	*BaseTx      `serialize:"true" json:"baseTx"`
	RewardAmount uint64 `serialize:"true" json:"reward"`
	EndTime      uint64 `serialize:"true" json:"endTime"`
}

func (c *ClaimTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()

	//sender is staker
	//ok, stakerType, _ := samaState.IsStaker(t.Sender)
	//if !ok {
	//	return fmt.Errorf("sender must staker")
	//}
	ok, claimerType, _ := samaState.CheckClaimAddress(t.Sender)
	if !ok {
		return fmt.Errorf("sender must staker or foundation")
	}

	lastClaimTime, _ := samaState.GetLastClaimTime(t.Sender)
	if uint64(time.Now().Unix())-lastClaimTime < Seconds7Day {
		return fmt.Errorf("time interval need > 7 days")
	}

	createTime := samaState.GetChainCreateTime()
	endTime := ((uint64(time.Now().Unix())-createTime)/SecondsDay)*SecondsDay + createTime

	if endTime != c.EndTime && endTime != c.EndTime+1 {
		return fmt.Errorf("end time err")
	}
	base, merit, yield, err := samaState.CalcReward(claimerType, t.Sender, c.EndTime)
	if err != nil {
		return err
	}
	totalReawrd := base + merit + yield
	if c.RewardAmount != totalReawrd {
		return fmt.Errorf("reward amount is err")
	}

	if _, err := ModifyBalance(t.Database, t.Sender, true, uint64(totalReawrd)); err != nil {
		return err
	}

	switch claimerType {
	case 0:
		err = samaState.UpdateFoundationReward(t.Database, t.Sender, t.TxID, c.EndTime)
		if err != nil {
			return err
		}
	default:
		err = samaState.UpdateStakerReward(t.Database, claimerType, t.Sender, t.TxID, c.EndTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ClaimTx) FeeUnits(g *Genesis) uint64 {
	return c.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(c.Value)))
}

func (c *ClaimTx) LoadUnits(g *Genesis) uint64 {
	return c.FeeUnits(g)
}

func (c *ClaimTx) Copy() UnsignedTransaction {
	return &ClaimTx{
		BaseTx:       c.BaseTx.Copy(),
		RewardAmount: c.RewardAmount,
		EndTime:      c.EndTime,
	}
}

func (c *ClaimTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		c.Magic, Claim,
		[]tdata.Type{
			{Name: tdReward, Type: tdUint64},
			{Name: tdEndTime, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdReward:  strconv.FormatUint(c.RewardAmount, 10),
			tdEndTime: strconv.FormatUint(c.EndTime, 10),
			tdPrice:   strconv.FormatUint(c.Price, 10),
			tdBlockID: c.BlockID.String(),
		},
	)
}

func (c *ClaimTx) Activity() *Activity {
	return &Activity{
		Typ:          Claim,
		RewardAmount: c.RewardAmount,
		EndTime:      c.EndTime,
	}
}
