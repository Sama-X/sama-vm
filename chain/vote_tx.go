// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

var _ UnsignedTransaction = &VoteTx{}

type VoteTx struct {
	*BaseTx  `serialize:"true" json:"baseTx"`
	ActionID ids.ShortID `serialize:"true" json:"actionID"`
}

func (v *VoteTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()
	ok, err := samaState.IsRoute(t.Sender)
	if err != nil {
		return err
	}
	if !ok {
		sroot := samaState.GetRootAddress()
		if t.Sender != common.HexToAddress(sroot) {
			return fmt.Errorf("sender is not route node %s %s", t.Sender.String(), sroot)
		}
		routeNum, _, _ := samaState.GetStakersNum()
		if routeNum > 3 {
			return fmt.Errorf("sender do not have permission")
		}
	}
	action, exist, err := samaState.GetActionMeta(v.ActionID)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("not found action")
	}
	for _, voter := range action.Voters {
		if voter == t.Sender {
			return fmt.Errorf("voter is exist")
		}
	}

	if uint64(time.Now().Unix()) > action.EndTime {
		return fmt.Errorf("action timeout")
	}

	pmate := new(ActionMeta)
	*pmate = *action
	pmate.TxIDs = append(pmate.TxIDs, t.TxID)
	pmate.Voters = append(pmate.Voters, t.Sender)
	samaState.PutAction(t.Database, v.ActionID, pmate)
	return nil
}

func (v *VoteTx) FeeUnits(genesis *Genesis) uint64 {
	return v.BaseTx.FeeUnits(genesis) //+ valueUnits(v, uint64(len(v.Value)))
}

func (v *VoteTx) LoadUnits(genesis *Genesis) uint64 {
	return v.FeeUnits(genesis)
}

func (v *VoteTx) Copy() UnsignedTransaction {
	actionID := ids.ShortID{}
	copy(actionID[:], v.ActionID[:])
	return &VoteTx{
		BaseTx:   v.BaseTx.Copy(),
		ActionID: actionID,
	}
}

func (v *VoteTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		v.Magic, Govern,
		[]tdata.Type{
			{Name: tdActionID, Type: tdString},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdActionID: v.ActionID.String(),
			tdPrice:    strconv.FormatUint(v.Price, 10),
			tdBlockID:  v.BlockID.String(),
		},
	)
}

func (v *VoteTx) Activity() *Activity {
	return &Activity{
		Typ:      Govern,
		ActionID: v.ActionID,
	}
}
