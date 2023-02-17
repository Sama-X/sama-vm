// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

const (
	powTypeSer   = serPrefix
	powTypeRoute = routePrefix
)

func PrefixPowKey(powType byte, address common.Address) (k []byte) {
	k = make([]byte, 4+common.AddressLength)
	k[0] = powPrefix
	k[1] = ByteDelimiter
	k[2] = powType
	k[3] = ByteDelimiter
	copy(k[4:], address[:])
	return
}

func basePowPrefix(powType byte) (k []byte) {
	k = make([]byte, 4)
	k[0] = powPrefix
	k[1] = ByteDelimiter
	k[2] = powType
	k[3] = ByteDelimiter
	return
}

var _ PowState = &powState{}
var powTypes = []byte{powTypeSer, powTypeRoute}

type ProofMeta struct {
	Netflow    uint64         `serialize:"true" json:"netflow"`
	WorkTime   uint64         `serialize:"true" json:"workTime"`
	Miner      common.Address `serialize:"true" json:"miner"`
	TxID       ids.ID         `serialize:"true" json:"txId"`
	UpdateTime uint64         `serialize:"true" json:"updateTime"`
}

type PowMeta struct {
	PowType        uint64         `serialize:"true" json:"powType"`
	TotalTime      uint64         `serialize:"true" json:"totalTime"`
	Totalflow      uint64         `serialize:"true" json:"totalflow"`
	LastUpdateTime uint64         `serialize:"true" json:"lastUpdateTime"`
	LastUpdateTXID ids.ID         `serialize:"true" json:"lastUpdateTxId"`
	Miner          common.Address `serialize:"true" json:"miner"`
}

type PowState interface {
	GetCurPow(powType byte) (map[common.Address]*PowMeta, error)
	GetPendingPow(powType byte) (map[common.Address]*PowMeta, map[common.Address]*PowMeta, error)
	GetPowMeta(powType byte, address common.Address) (*PowMeta, bool, error)

	PutPow(db database.Database, powType byte, pmeta *ProofMeta) error

	GetPows(powType byte) ([]*PowMeta, error)

	DelPow(db database.Database, powType byte, address common.Address) error
	ReloadPows(db database.Database) error
	CachePowsCommit() error
	CachePowsAbort() error

	TotalPowTime(powType byte) (uint64, error)
}

type powState struct {
	updateTime uint64

	curSer   map[common.Address]*PowMeta
	curRoute map[common.Address]*PowMeta

	pendingRouteAdd map[common.Address]*PowMeta
	pendingSerAdd   map[common.Address]*PowMeta

	pendingRouteDel map[common.Address]*PowMeta
	pendingSerDel   map[common.Address]*PowMeta
}

func NewPowState(db database.Database) (*powState, error) {
	s := &powState{
		updateTime: 0,

		curSer:   make(map[common.Address]*PowMeta),
		curRoute: make(map[common.Address]*PowMeta),

		pendingRouteAdd: make(map[common.Address]*PowMeta),
		pendingSerAdd:   make(map[common.Address]*PowMeta),

		pendingRouteDel: make(map[common.Address]*PowMeta),
		pendingSerDel:   make(map[common.Address]*PowMeta),
	}
	err := s.ReloadPows(db)
	return s, err
}
func (f *powState) GetPendingPow(powType byte) (map[common.Address]*PowMeta, map[common.Address]*PowMeta, error) {
	switch powType {
	case powTypeSer:
		return f.pendingRouteAdd, f.pendingRouteDel, nil
	case powTypeRoute:
		return f.pendingSerAdd, f.pendingSerDel, nil
	default:
		return nil, nil, ErrIDErr
	}
}

func (f *powState) GetCurPow(powType byte) (map[common.Address]*PowMeta, error) {
	switch powType {
	case powTypeSer:
		return f.curSer, nil
	case powTypeRoute:
		return f.curRoute, nil
	default:
		return nil, ErrIDErr
	}
}

func (f *powState) GetPowMeta(powType byte, address common.Address) (*PowMeta, bool, error) {
	curMap, _ := f.GetCurPow(powType)
	pmeta, ok := curMap[address]
	if !ok {
		return nil, false, nil
	}
	return pmeta, true, nil
}

func (f *powState) PutPow(db database.Database, powType byte, proof *ProofMeta) error {
	k := PrefixPowKey(powType, proof.Miner)
	pendingAdd, pendingDel, _ := f.GetPendingPow(powType)
	delete(pendingDel, proof.Miner)
	pendingMeta, ok := pendingAdd[proof.Miner]
	if ok {
		pendingMeta.Totalflow += proof.Netflow
		pendingMeta.TotalTime += proof.WorkTime
		pendingMeta.LastUpdateTXID = proof.TxID
		pendingMeta.LastUpdateTime = proof.UpdateTime

		pvmeta, err := Marshal(pendingMeta)
		if err != nil {
			return err
		}
		return db.Put(k, pvmeta)
	} else {
		pmeta := new(PowMeta)
		curMap, _ := f.GetCurPow(powType)
		curMeta, ok := curMap[proof.Miner]
		if ok {
			if curMeta.PowType != uint64(powType) {
				return fmt.Errorf("pow type err")
			}
			pmeta.Totalflow = curMeta.Totalflow + proof.Netflow
			pmeta.TotalTime = curMeta.TotalTime + proof.WorkTime
		} else {
			pmeta.Totalflow = proof.Netflow
			pmeta.TotalTime = proof.WorkTime
		}
		pmeta.PowType = uint64(powType)
		pmeta.LastUpdateTXID = proof.TxID
		pmeta.LastUpdateTime = proof.UpdateTime
		pmeta.Miner = proof.Miner
		pendingAdd[proof.Miner] = pmeta
		pvmeta, err := Marshal(pmeta)
		if err != nil {
			return err
		}
		return db.Put(k, pvmeta)
	}
}

func (f *powState) TotalPowTime(powType byte) (uint64, error) {
	total := uint64(0)
	curMap, _ := f.GetCurPow(powType)
	for _, pow := range curMap {
		total += pow.TotalTime
	}
	return total, nil
}

func (f *powState) GetPows(powType byte) ([]*PowMeta, error) {
	pows := []*PowMeta(nil)
	curMap, _ := f.GetCurPow(powType)
	for _, pow := range curMap {
		pows = append(pows, pow)
	}
	return pows, nil
}

func (f *powState) DelPow(db database.Database, powType byte, address common.Address) error {
	pendingAdd, pendingDel, _ := f.GetPendingPow(powType)
	delete(pendingAdd, address)
	pendingDel[address] = &PowMeta{}

	k := PrefixPowKey(powType, address)
	return db.Delete(k)
}

func (f *powState) ReloadPows(db database.Database) error {
	for _, powType := range powTypes {
		basePrefix := basePowPrefix(powType)
		cursor := db.NewIteratorWithPrefix(basePrefix)
		defer cursor.Release()
		for cursor.Next() {
			nmeta := cursor.Value()

			pmeta := new(PowMeta)
			if _, err := Unmarshal(nmeta, pmeta); err != nil {
				return err
			}
			curMap, _ := f.GetCurPow(powType)
			curMap[pmeta.Miner] = pmeta
		}
	}
	return nil
}

func (f *powState) CachePowsCommit() error {
	for _, powType := range powTypes {
		pendingAdd, pendingDel, _ := f.GetPendingPow(powType)
		curMap, _ := f.GetCurPow(powType)

		for addr, pendingPow := range pendingAdd {
			delete(curMap, addr)
			curMap[addr] = pendingPow
		}
		for addr := range pendingDel {
			delete(curMap, addr)
		}
		for a := range pendingAdd {
			delete(pendingAdd, a)
		}
		for a := range pendingDel {
			delete(pendingDel, a)
		}
	}
	return nil
}

func (f *powState) CachePowsAbort() error {
	for _, powType := range powTypes {
		pendingAdd, pendingDel, _ := f.GetPendingPow(powType)
		for a := range pendingAdd {
			delete(pendingAdd, a)
		}
		for a := range pendingDel {
			delete(pendingDel, a)
		}
	}
	return nil
}
