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

var _ UnsignedTransaction = &GovernTx{}

type GovernTx struct {
	*BaseTx  `serialize:"true" json:"baseTx"`
	ActionID ids.ShortID `serialize:"true" json:"actionID"`
}

func (g *GovernTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()

	action, exist, err := samaState.GetActionMeta(g.ActionID)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("action not found")
	}

	//if uint64(time.Now().Unix()) < action.EndTime {
	//	return fmt.Errorf("need > 7 days")
	//}

	if uint64(time.Now().Unix()) > action.EndTime+Seconds7Day {
		return fmt.Errorf("need < 14 days")
	}

	isBeConfirmed := false
	votersNum := len(action.Voters)
	routeNum, _, _ := samaState.GetStakersNum()
	if routeNum < 3 {
		sroot := samaState.GetRootAddress()
		for _, voter := range action.Voters {
			if voter == common.HexToAddress(sroot) {
				isBeConfirmed = true
				break
			}
		}
	} else {
		if votersNum >= routeNum*2/3 {
			isBeConfirmed = true
		}
	}
	if !isBeConfirmed {
		return fmt.Errorf("action not be confirmed")
	}
	key := action.Key
	newValue := action.NewValue
	switch action.ActionType {
	case actionTypeModifySysParam, actionTypeModifyFoundation:
		err = samaState.ModifyParams(t.Database, key, newValue, t.TxID, t.BlockTime)
	case actionTypeAddUserType, actionTypeModifyUserType:
		userType, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			return err
		}
		userFee, err := strconv.ParseUint(newValue, 10, 64)
		if err != nil {
			return err
		}
		err = samaState.AddUserType(t.Database, &UserType{
			TypeID:   userType,
			FeeUnits: userFee,
		})
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("action type err")
	}

	return err
}

func (g *GovernTx) FeeUnits(genesis *Genesis) uint64 {
	return g.BaseTx.FeeUnits(genesis) //+ valueUnits(g, uint64(len(g.Value)))
}

func (g *GovernTx) LoadUnits(genesis *Genesis) uint64 {
	return g.FeeUnits(genesis)
}

func (g *GovernTx) Copy() UnsignedTransaction {
	return &GovernTx{
		BaseTx:   g.BaseTx.Copy(),
		ActionID: g.ActionID,
	}
}

func (g *GovernTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		g.Magic, Govern,
		[]tdata.Type{
			{Name: tdActionID, Type: tdString},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdActionID: g.ActionID.String(),
			tdPrice:    strconv.FormatUint(g.Price, 10),
			tdBlockID:  g.BlockID.String(),
		},
	)
}

func (g *GovernTx) Activity() *Activity {
	return &Activity{
		Typ:      Govern,
		ActionID: g.ActionID,
	}
}
