// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ava-labs/avalanchego/ids"
)

var _ UnsignedTransaction = &WithdrawnTx{}

type WithdrawnTx struct {
	*BaseTx  `serialize:"true" json:"baseTx"`
	ActionID ids.ShortID `serialize:"true" json:"actionID"`
}

func (w *WithdrawnTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()
	ok, err := samaState.IsRoute(t.Sender)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("sender is not route node")
	}
	pmate, exist, err := samaState.GetActionMeta(w.ActionID)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("not found action")
	}

	for i, voter := range pmate.Voters {
		if voter == t.Sender {
			pmate.Voters = append(pmate.Voters[:i], pmate.Voters[i+1:]...)
			break
		}
	}
	pmate.TxIDs = append(pmate.TxIDs, t.TxID)
	samaState.PutAction(t.Database, w.ActionID, pmate)
	return nil
}

func (w *WithdrawnTx) FeeUnits(genesis *Genesis) uint64 {
	return w.BaseTx.FeeUnits(genesis) //+ valueUnits(w, uint64(len(w.Value)))
}

func (w *WithdrawnTx) LoadUnits(genesis *Genesis) uint64 {
	return w.FeeUnits(genesis)
}

func (w *WithdrawnTx) Copy() UnsignedTransaction {
	actionID := ids.ShortID{}
	copy(actionID[:], w.ActionID[:])
	return &WithdrawnTx{
		BaseTx:   w.BaseTx.Copy(),
		ActionID: actionID,
	}
}

func (w *WithdrawnTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		w.Magic, Withdrawn,
		[]tdata.Type{
			{Name: tdActionID, Type: tdString},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdActionID: w.ActionID.String(),
			tdPrice:    strconv.FormatUint(w.Price, 10),
			tdBlockID:  w.BlockID.String(),
		},
	)
}

func (w *WithdrawnTx) Activity() *Activity {
	return &Activity{
		Typ:      Withdrawn,
		ActionID: w.ActionID,
	}
}
