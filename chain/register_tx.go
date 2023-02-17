// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var _ UnsignedTransaction = &RegisterTx{}

type RegisterTx struct {
	*BaseTx    `serialize:"true" json:"baseTx"`
	ActionID   ids.ShortID    `serialize:"true" json:"actionID"`
	StakerType uint64         `serialize:"true" json:"stakerType"`
	Country    string         `serialize:"true" json:"country"`
	LocalIP    string         `serialize:"true" json:"localIP"`
	PublicIP   string         `serialize:"true" json:"publicIP"`
	MinPort    uint64         `serialize:"true" json:"minPort"`
	MaxPort    uint64         `serialize:"true" json:"maxPort"`
	CheckPort  uint64         `serialize:"true" json:"checkPort"`
	WorkKey    string         `serialize:"true" json:"workKey"`
	StakerAddr common.Address `serialize:"true" json:"stakeAddress"`
}

func IsRegistered(d *DetailMeta, r *RegisterTx) bool {
	if d.PublicIP == r.PublicIP && d.WorkKey == r.WorkKey &&
		d.StakerType == r.StakerType {
		return true
	}
	return false
}

func (r *RegisterTx) Execute(t *TransactionContext) error {
	if bytes.Equal(r.StakerAddr[:], zeroAddress[:]) {
		return ErrNonActionable
	}
	pbkb, err := hex.DecodeString(r.WorkKey)
	if err != nil {
		return fmt.Errorf("work key err %w", err)
	}
	pbk, err := crypto.UnmarshalPubkey(pbkb)
	if err != nil {
		return fmt.Errorf("unmarshalPubkey err %w", err)
	}
	addr := crypto.PubkeyToAddress(*pbk)

	samaState := t.vm.SamaState()
	ok, _, _ := samaState.IsStaker(r.StakerAddr)
	if ok {
		return fmt.Errorf("have already been staker")
	}
	pmate, exist, err := samaState.GetDetailMeta(addr)
	if err != nil {
		return err
	}
	if exist {
		if pmate.StakerType != r.StakerType {
			return fmt.Errorf("staker type err")
		} else {
			if IsRegistered(pmate, r) {
				return fmt.Errorf("is repeated")
			}
		}
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
	err = samaState.PutAction(t.Database, r.ActionID, &ActionMeta{
		ActionID:   r.ActionID,
		ActionType: uint64(actionTypeAddStaker),
		StartTime:  t.BlockTime,
		EndTime:    t.BlockTime + Seconds7Day,
		Key:        r.StakerAddr.Hex(),
		TxIDs:      txIDs,
		Voters:     voters,
	})
	if err != nil {
		return err
	}
	err = samaState.UpdateNodeParams(t.Database, &DetailMeta{
		StakerType:     r.StakerType,
		Country:        r.Country,
		LocalIP:        r.LocalIP,
		MinPort:        r.MinPort,
		MaxPort:        r.MaxPort,
		PublicIP:       r.PublicIP,
		CheckPort:      r.CheckPort,
		WorkKey:        r.WorkKey,
		TxID:           t.TxID,
		LastUpdateTime: t.BlockTime,
		WorkAddress:    addr,
		StakeAddress:   r.StakerAddr,
	})
	return err
}

func (r *RegisterTx) FeeUnits(g *Genesis) uint64 {
	return r.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(r.Value)))
}

func (r *RegisterTx) LoadUnits(g *Genesis) uint64 {
	return r.FeeUnits(g)
}

func (r *RegisterTx) Copy() UnsignedTransaction {
	actionID := ids.ShortID{}
	copy(actionID[:], r.ActionID[:])
	return &RegisterTx{
		BaseTx:     r.BaseTx.Copy(),
		ActionID:   actionID,
		StakerType: r.StakerType,
		Country:    r.Country,
		WorkKey:    r.WorkKey,
		LocalIP:    r.LocalIP,
		MinPort:    r.MinPort,
		MaxPort:    r.MaxPort,
		PublicIP:   r.PublicIP,
		CheckPort:  r.CheckPort,
		StakerAddr: r.StakerAddr,
	}
}

func (r *RegisterTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		r.Magic, Register,
		[]tdata.Type{
			{Name: tdActionID, Type: tdString},
			{Name: tdStakerType, Type: tdUint64},
			{Name: tdCountry, Type: tdString},
			{Name: tdWorkKey, Type: tdString},
			{Name: tdLocalIP, Type: tdString},
			{Name: tdMinPort, Type: tdUint64},
			{Name: tdMaxPort, Type: tdUint64},
			{Name: tdPublicIP, Type: tdString},
			{Name: tdCheckPort, Type: tdUint64},
			{Name: tdStakeAddr, Type: tdAddress},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdActionID:   r.ActionID.String(),
			tdStakerType: strconv.FormatUint(r.StakerType, 10),
			tdCountry:    r.Country,
			tdWorkKey:    r.WorkKey,
			tdLocalIP:    r.LocalIP,
			tdMinPort:    strconv.FormatUint(r.MinPort, 10),
			tdMaxPort:    strconv.FormatUint(r.MaxPort, 10),
			tdPublicIP:   r.PublicIP,
			tdCheckPort:  strconv.FormatUint(r.CheckPort, 10),
			tdStakeAddr:  r.StakerAddr.Hex(),
			tdPrice:      strconv.FormatUint(r.Price, 10),
			tdBlockID:    r.BlockID.String(),
		},
	)
}

func (r *RegisterTx) Activity() *Activity {
	return &Activity{
		Typ:        Register,
		ActionID:   r.ActionID,
		StakerType: r.StakerType,
		Country:    r.Country,
		WorkKey:    r.WorkKey,
		LocalIP:    r.LocalIP,
		MinPort:    r.MinPort,
		MaxPort:    r.MaxPort,
		PublicIP:   r.PublicIP,
		CheckPort:  r.CheckPort,
		StakerAddr: r.StakerAddr.Hex(),
	}
}
