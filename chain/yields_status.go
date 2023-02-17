// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
)

func PrefixYieldsKey() (k []byte) {
	k = make([]byte, 2)
	k[0] = yieldsPrefix
	k[1] = ByteDelimiter
	return
}

var _ YieldsState = &yieldsState{}

type YieldMeta struct {
	Total         uint64 `serialize:"true" json:"total"`
	Undistributed uint64 `serialize:"true" json:"undistributed"`
	LastOprTime   uint64 `serialize:"true" json:"lastOprTime"`
	LastOprTXID   ids.ID `serialize:"true" json:"lastOprTxId"`
}

type YieldsState interface {
	GetChainYields() uint64
	ModifyYields(db database.Database, yield uint64, txID ids.ID, blkTime uint64) error
	ReloadYields(db database.Database) error
	CacheYieldsCommit() error
	CacheYieldsAbort() error
}

type yieldsState struct {
	curYields     *YieldMeta
	pendingYields *YieldMeta
}

func NewYieldsState(db database.Database) (*yieldsState, error) {
	s := &yieldsState{
		curYields: &YieldMeta{
			Total:         0,
			Undistributed: 0,
			LastOprTime:   0,
		},
		pendingYields: &YieldMeta{
			Total:         0,
			Undistributed: 0,
			LastOprTime:   0,
		},
	}
	err := s.ReloadYields(db)
	if err != nil {
		if err != database.ErrNotFound {
			return nil, err
		}
	}
	return s, nil
}

func (y *yieldsState) GetChainYields() uint64 {
	return y.curYields.Total
}

func (y *yieldsState) ModifyYields(db database.Database, yield uint64, txID ids.ID, blkTime uint64) error {
	y.pendingYields.Total = y.curYields.Total + yield
	if yield == 0 {
		y.pendingYields.Undistributed = 0
	} else {
		y.pendingYields.Undistributed = y.curYields.Undistributed + yield
	}
	y.pendingYields.LastOprTXID = txID
	y.pendingYields.LastOprTime = blkTime

	k := PrefixYieldsKey()
	pvmeta, err := Marshal(y.pendingYields)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (y *yieldsState) ReloadYields(db database.Database) error {
	k := PrefixYieldsKey()
	ymeta, err := db.Get(k)
	if err != nil {
		return err
	}
	if _, err := Unmarshal(ymeta, y.curYields); err != nil {
		return err
	}
	return nil
}

func (y *yieldsState) CacheYieldsCommit() error {
	if y.pendingYields.Total != 0 {
		*y.curYields = *y.pendingYields
		y.pendingYields.Total = 0
	}
	return nil
}

func (y *yieldsState) CacheYieldsAbort() error {
	y.pendingYields.Total = 0
	return nil
}
