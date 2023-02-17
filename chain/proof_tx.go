// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ethereum/go-ethereum/common"
)

const (
	minFlow = 1
	maxFlow = 1000
	minTime = 1
	maxTime = 60
)

var _ UnsignedTransaction = &ProofTx{}

type ProofTx struct {
	*BaseTx   `serialize:"true" json:"baseTx"`
	Netflow   uint64         `serialize:"true" json:"netflow"`
	StartTime uint64         `serialize:"true" json:"startTime"`
	EndTime   uint64         `serialize:"true" json:"endTime"`
	Ser       common.Address `serialize:"true" json:"ser"`
}

func (p *ProofTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()
	ok, powType, err := samaState.IsValidWorkAddress(t.Sender)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("sender must route or ser node")
	}

	interval := p.EndTime - p.StartTime
	switch {
	case p.Netflow < minFlow:
		return fmt.Errorf("netflow too small")
	case p.Netflow > maxFlow:
		return fmt.Errorf("netflow too big")
	case p.StartTime > p.EndTime:
		return fmt.Errorf("start time > endtime")
	case p.StartTime > uint64(time.Now().Unix())-50:
		return fmt.Errorf("start time err")
	case p.EndTime > uint64(time.Now().Unix()):
		return fmt.Errorf("endtime err")
	case interval < minTime:
		return fmt.Errorf("interval too small")
	case interval > maxTime:
		return fmt.Errorf("interval to big")
	}
	err = samaState.PutPow(t.Database, powType, &ProofMeta{
		Netflow:    p.Netflow,
		WorkTime:   p.EndTime - p.StartTime,
		Miner:      t.Sender,
		TxID:       t.TxID,
		UpdateTime: t.BlockTime,
	})

	return err
}

func (p *ProofTx) FeeUnits(g *Genesis) uint64 {
	return 0 //p.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(p.Value)))
}

func (p *ProofTx) LoadUnits(g *Genesis) uint64 {
	return 0 //p.FeeUnits(g)
}

func (p *ProofTx) Copy() UnsignedTransaction {
	return &ProofTx{
		BaseTx:    p.BaseTx.Copy(),
		Netflow:   p.Netflow,
		StartTime: p.StartTime,
		EndTime:   p.EndTime,
		Ser:       p.Ser,
	}
}

func (p *ProofTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		p.Magic, Proof,
		[]tdata.Type{
			{Name: tdSer, Type: tdAddress},
			{Name: tdStartTime, Type: tdUint64},
			{Name: tdEndTime, Type: tdUint64},
			{Name: tdNetflow, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdSer:       p.Ser.Hex(),
			tdStartTime: strconv.FormatUint(p.StartTime, 10),
			tdEndTime:   strconv.FormatUint(p.EndTime, 10),
			tdNetflow:   strconv.FormatUint(p.Netflow, 10),

			tdPrice:   strconv.FormatUint(p.Price, 10),
			tdBlockID: p.BlockID.String(),
		},
	)
}

func (p *ProofTx) Activity() *Activity {
	return &Activity{
		Typ:       Proof,
		StartTime: p.StartTime,
		EndTime:   p.EndTime,
		Netflow:   p.Netflow,
		Address:   p.Ser.Hex(),
	}
}
