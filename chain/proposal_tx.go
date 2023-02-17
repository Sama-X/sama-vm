// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

var _ UnsignedTransaction = &ProposalTx{}

type ProposalTx struct {
	*BaseTx    `serialize:"true" json:"baseTx"`
	ActionID   ids.ShortID `serialize:"true" json:"actionID"`
	StartTime  uint64      `serialize:"true" json:"startTime"`
	EndTime    uint64      `serialize:"true" json:"endTime"`
	ActionType uint64      `serialize:"true" json:"actionType"`
	Key        string      `serialize:"true" json:"key"`
	NewValue   string      `serialize:"true" json:"newValue"`
}

func (p *ProposalTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()
	_, exist, _ := samaState.GetActionMeta(p.ActionID)
	if exist {
		return fmt.Errorf("action ID exist")
	}
	switch p.ActionType {
	//case actionTypeAddStaker:

	case actionTypeModifySysParam, actionTypeModifyFoundation:
		err := samaState.CompCurParam(p.Key, p.NewValue)
		if err != nil {
			return err
		}
	case actionTypeAddUserType:
		newType, err := strconv.ParseUint(p.Key, 10, 64)
		if err != nil {
			return err
		}
		exist := samaState.CheckUserType(newType)
		if exist {
			return fmt.Errorf("user type exist")
		}
	case actionTypeModifyUserType:
		userType, err := strconv.ParseUint(p.Key, 10, 64)
		if err != nil {
			return err
		}

		userFee, err := strconv.ParseUint(p.NewValue, 10, 64)
		if err != nil {
			return err
		}
		exist := samaState.CheckUserType(userType)
		if !exist {
			return fmt.Errorf("user type not exist")
		}
		eq := samaState.CheckFee(userType, userFee)
		if eq {
			return fmt.Errorf("equal")
		}
	default:
		return fmt.Errorf("action type not exist")
	}

	err := samaState.ProposalRepeat(p.ActionID, p.Key, p.NewValue)
	if err != nil {
		return err
	}
	txIDs := []ids.ID{}
	txIDs = append(txIDs, t.TxID)
	voters := []common.Address{}

	routeNum, _, _ := samaState.GetStakersNum()
	sroot := samaState.GetRootAddress()

	if t.Sender == common.HexToAddress(sroot) && routeNum < 3 {
		voters = append(voters, t.Sender)
	} else {
		ok, _ := samaState.IsRoute(t.Sender)
		if ok {
			voters = append(voters, t.Sender)
		}
	}
	err = samaState.PutAction(t.Database, p.ActionID, &ActionMeta{
		ActionID:   p.ActionID,
		ActionType: p.ActionType,
		StartTime:  t.BlockTime,
		Key:        p.Key,
		NewValue:   p.NewValue,
		EndTime:    t.BlockTime + Seconds7Day,
		TxIDs:      txIDs,
		Voters:     voters,
	})

	return err
}

func (p *ProposalTx) FeeUnits(genesis *Genesis) uint64 {
	return p.BaseTx.FeeUnits(genesis) //+ valueUnits(p, uint64(len(p.Value)))
}

func (p *ProposalTx) LoadUnits(genesis *Genesis) uint64 {
	return p.FeeUnits(genesis)
}

func (p *ProposalTx) Copy() UnsignedTransaction {
	return &ProposalTx{
		BaseTx:    p.BaseTx.Copy(),
		ActionID:  p.ActionID,
		StartTime: p.StartTime,
		EndTime:   p.EndTime,
		Key:       p.Key,
		NewValue:  p.NewValue,
	}
}

func (p *ProposalTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		p.Magic, Proposal,
		[]tdata.Type{
			{Name: tdActionID, Type: tdString},
			{Name: tdKey, Type: tdString},
			{Name: tdNewValue, Type: tdString},
			{Name: tdStartTime, Type: tdUint64},
			{Name: tdEndTime, Type: tdUint64},
			{Name: tdActionType, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdActionID:   p.ActionID.String(),
			tdKey:        p.Key,
			tdNewValue:   p.NewValue,
			tdStartTime:  strconv.FormatUint(p.StartTime, 10),
			tdEndTime:    strconv.FormatUint(p.EndTime, 10),
			tdActionType: strconv.FormatUint(p.ActionType, 10),
			tdPrice:      strconv.FormatUint(p.Price, 10),
			tdBlockID:    p.BlockID.String(),
		},
	)
}

func (p *ProposalTx) Activity() *Activity {
	return &Activity{
		Typ:        Proposal,
		ActionID:   p.ActionID,
		StartTime:  p.StartTime,
		EndTime:    p.EndTime,
		ActionType: p.ActionType,
		Key:        p.Key,
		Value:      p.NewValue,
	}
}
