// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ethereum/go-ethereum/common"
)

var _ UnsignedTransaction = &AddUserTx{}

type AddUserTx struct {
	*BaseTx     `serialize:"true" json:"baseTx"`
	StartTime   uint64         `serialize:"true" json:"startTime"`
	EndTime     uint64         `serialize:"true" json:"endTime"`
	PayAmount   uint64         `serialize:"true" json:"payAmount"`
	Connections uint64         `serialize:"true" json:"connections"`
	UserType    uint64         `serialize:"true" json:"userType"`
	Address     common.Address `serialize:"true" json:"address"`
}

func (a *AddUserTx) Execute(t *TransactionContext) error {
	switch {
	case a.StartTime > a.EndTime:
		return fmt.Errorf("start time > endtime")
	case a.EndTime-a.StartTime > 31*24*60*60:
		return fmt.Errorf("time > 31 days")
	case a.EndTime-a.StartTime < 28*24*60*60:
		return fmt.Errorf("time < 28 days")
	case a.StartTime < uint64(time.Now().Unix())-60:
		return fmt.Errorf("start time err")
	}

	if bytes.Equal(a.Address[:], zeroAddress[:]) {
		return ErrNonActionable
	}
	samaState := t.vm.SamaState()

	ok := samaState.CheckUserType(a.UserType)
	if !ok {
		return fmt.Errorf("user type err")
	}

	ok, err := samaState.CheckPayAmount(t.Database, a.UserType, a.PayAmount, a.StartTime, a.EndTime)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("pay amount check fail")
	}
	_, exists, _ := samaState.GetUserMeta(t.Database, a.Address)
	if exists {
		//checkEndTime
		return fmt.Errorf("ueser have already exist ")
	}

	if _, err := ModifyBalance(t.Database, t.Sender, false, a.PayAmount); err != nil {
		return err
	}

	return samaState.DealAddUserTx(t.Database, t.TxID, t.BlockTime, &UserMeta{
		StartTime:   a.StartTime,
		EndTime:     a.EndTime,
		UserType:    a.UserType,
		Connections: a.Connections,
		PayAmount:   a.PayAmount,
		Address:     a.Address,
	})

}

func (c *AddUserTx) FeeUnits(g *Genesis) uint64 {
	return c.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(c.Value)))
}

func (c *AddUserTx) LoadUnits(g *Genesis) uint64 {
	return c.FeeUnits(g)
}

func (c *AddUserTx) Copy() UnsignedTransaction {
	user := make([]byte, common.AddressLength)
	copy(user, c.Address[:])
	return &AddUserTx{
		BaseTx:      c.BaseTx.Copy(),
		StartTime:   c.StartTime,
		EndTime:     c.EndTime,
		PayAmount:   c.PayAmount,
		UserType:    c.UserType,
		Connections: c.Connections,
		Address:     common.BytesToAddress(user),
	}
}

func (c *AddUserTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		c.Magic, AddUser,
		[]tdata.Type{
			{Name: tdAddress, Type: tdAddress},
			{Name: tdStartTime, Type: tdUint64},
			{Name: tdEndTime, Type: tdUint64},
			{Name: tdAmount, Type: tdUint64},
			{Name: tdUserType, Type: tdUint64},

			{Name: tdConnections, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdAddress:     c.Address.Hex(),
			tdStartTime:   strconv.FormatUint(c.StartTime, 10),
			tdEndTime:     strconv.FormatUint(c.EndTime, 10),
			tdAmount:      strconv.FormatUint(c.PayAmount, 10),
			tdUserType:    strconv.FormatUint(c.UserType, 10),
			tdConnections: strconv.FormatUint(c.Connections, 10),
			tdPrice:       strconv.FormatUint(c.Price, 10),
			tdBlockID:     c.BlockID.String(),
		},
	)
}

func (c *AddUserTx) Activity() *Activity {
	return &Activity{
		Typ:         AddUser,
		StartTime:   c.StartTime,
		EndTime:     c.EndTime,
		PayAmount:   c.PayAmount,
		UserType:    c.UserType,
		Connections: c.Connections,
		Address:     c.Address.Hex(),
	}
}
