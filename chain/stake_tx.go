// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ethereum/go-ethereum/common"
)

var _ UnsignedTransaction = &StakeTx{}

type StakeTx struct {
	*BaseTx     `serialize:"true" json:"baseTx"`
	StakerType  uint64         `serialize:"true" json:"stakerType"`
	StakeAmount uint64         `serialize:"true" json:"stakeAmount"`
	StakerAddr  common.Address `serialize:"true" json:"stakerAddr"`
}

func (s *StakeTx) Execute(t *TransactionContext) error {
	g := t.Genesis
	switch {
	case s.StakerType != stakerTypeRoute && s.StakerType != stakerTypeSer && s.StakerType != stakerTypeValidator:
		return ErrStakerType
	case s.StakeAmount != g.RouteStake && s.StakeAmount != g.SerStake:
		return ErrStakeAmount
	}

	if bytes.Equal(s.StakerAddr[:], zeroAddress[:]) {
		return ErrNonActionable
	}
	samaState := t.vm.SamaState()

	exists, _, _ := samaState.IsStaker(s.StakerAddr)
	if exists {
		return fmt.Errorf("have already been staker")
	}

	ok, actionID, _ := samaState.IsBeConfirmed(actionTypeAddStaker, t.Sender.Hex())
	if !ok {
		return fmt.Errorf("no be confirmed")
	}
	if _, err := ModifyBalance(t.Database, t.Sender, false, s.StakeAmount); err != nil {
		return err
	}

	if _, err := ModifyStakeBalance(t.Database, t.Sender, true, s.StakeAmount); err != nil {
		return err
	}

	err := samaState.DealStakeTx(t.Database, &StakerMeta{
		TxID:        t.TxID,
		StakerType:  s.StakerType,
		StakerAddr:  s.StakerAddr,
		StakeAmount: s.StakeAmount,
		StakeTime:   t.BlockTime, //- SecondsDay*2, for test
	})
	if err != nil {
		return err
	}
	err = samaState.DelAction(t.Database, actionID)

	return err
}

func (s *StakeTx) FeeUnits(g *Genesis) uint64 {
	return s.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(s.Value)))
}

func (s *StakeTx) LoadUnits(g *Genesis) uint64 {
	return s.FeeUnits(g)
}

func (s *StakeTx) Copy() UnsignedTransaction {
	user := make([]byte, common.AddressLength)
	copy(user, s.StakerAddr[:])
	return &StakeTx{
		BaseTx:      s.BaseTx.Copy(),
		StakerType:  s.StakerType,
		StakeAmount: s.StakeAmount,
		StakerAddr:  common.BytesToAddress(user),
	}
}

func (s *StakeTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		s.Magic, Stake,
		[]tdata.Type{
			{Name: tdStakeAddr, Type: tdAddress},
			{Name: tdStakerType, Type: tdUint64},
			{Name: tdAmount, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdStakeAddr:  s.StakerAddr.Hex(),
			tdStakerType: strconv.FormatUint(s.StakerType, 10),
			tdAmount:     strconv.FormatUint(s.StakeAmount, 10),
			tdPrice:      strconv.FormatUint(s.Price, 10),
			tdBlockID:    s.BlockID.String(),
		},
	)
}

func (s *StakeTx) Activity() *Activity {
	return &Activity{
		Typ:         Stake,
		StakerType:  s.StakerType,
		StakeAmount: s.StakeAmount,
		StakerAddr:  s.StakerAddr.Hex(),
	}
}
