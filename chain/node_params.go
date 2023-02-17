// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

func PrefixDetailsKey(address common.Address) (k []byte) {
	k = make([]byte, 2+common.AddressLength)
	k[0] = detailsPrefix
	k[1] = ByteDelimiter
	copy(k[2:], address[:])
	return
}

func baseDetailsPrefix() (k []byte) {
	k = make([]byte, 2)
	k[0] = detailsPrefix
	k[1] = ByteDelimiter
	return
}

var _ DetailsState = &detailsState{}

type DetailMeta struct {
	StakerType     uint64         `serialize:"true" json:"stakerType"`
	Country        string         `serialize:"true" json:"country"`
	LocalIP        string         `serialize:"true" json:"localIP"`
	PublicIP       string         `serialize:"true" json:"publicIP"`
	MinPort        uint64         `serialize:"true" json:"minPort"`
	MaxPort        uint64         `serialize:"true" json:"maxPort"`
	CheckPort      uint64         `serialize:"true" json:"checkPort"`
	WorkKey        string         `serialize:"true" json:"workKey"`
	TxID           ids.ID         `serialize:"true" json:"txid"`
	LastUpdateTime uint64         `serialize:"true" json:"lastUpdateTime"`
	WorkAddress    common.Address `serialize:"true" json:"workAddress"`
	StakeAddress   common.Address `serialize:"true" json:"stakeAddress"`
}

type DetailsState interface {
	GetDetailMeta(address common.Address) (*DetailMeta, bool, error)
	PutDetail(db database.Database, address common.Address, pmeta *DetailMeta) error
	GetDetails() ([]*DetailMeta, error)
	DelDetail(db database.Database, address common.Address) error
	ReloadDetails(db database.Database) error
	CacheDetailsCommit() error
	CacheDetailsAbort() error
}

type detailsState struct {
	curNodes        map[common.Address]*DetailMeta
	pendingNodesAdd map[common.Address]*DetailMeta
	pendingNodesDel map[common.Address]*DetailMeta
}

func NewDetailstate(db database.Database) (*detailsState, error) {
	s := &detailsState{
		curNodes:        make(map[common.Address]*DetailMeta),
		pendingNodesAdd: make(map[common.Address]*DetailMeta),
		pendingNodesDel: make(map[common.Address]*DetailMeta),
	}

	err := s.ReloadDetails(db)
	return s, err
}

func (d *detailsState) GetDetailMeta(address common.Address) (*DetailMeta, bool, error) {
	pmeta, ok := d.curNodes[address]
	if !ok {
		return nil, false, nil
	}
	return pmeta, true, nil
}

func (d *detailsState) PutDetail(db database.Database, address common.Address, pmeta *DetailMeta) error {
	delete(d.pendingNodesDel, address)
	d.pendingNodesAdd[address] = pmeta

	k := PrefixDetailsKey(address)
	pvmeta, err := Marshal(pmeta)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (d *detailsState) GetDetails() ([]*DetailMeta, error) {
	details := []*DetailMeta(nil)
	for _, detail := range d.curNodes {
		details = append(details, detail)
	}
	return details, nil
}

func (d *detailsState) DelDetail(db database.Database, address common.Address) error {
	delete(d.pendingNodesAdd, address)
	d.pendingNodesDel[address] = &DetailMeta{}

	k := PrefixDetailsKey(address)
	return db.Delete(k)
}

func (d *detailsState) ReloadDetails(db database.Database) error {
	basePrefix := baseDetailsPrefix()
	cursor := db.NewIteratorWithPrefix(basePrefix)
	defer cursor.Release()
	for cursor.Next() {
		nmeta := cursor.Value()
		pmeta := new(DetailMeta)
		if _, err := Unmarshal(nmeta, pmeta); err != nil {
			return err
		}
		d.curNodes[pmeta.StakeAddress] = pmeta
	}
	return nil
}

func (d *detailsState) CacheDetailsCommit() error {
	for addr, pendingDetail := range d.pendingNodesAdd {
		delete(d.curNodes, addr)
		d.curNodes[addr] = pendingDetail
	}
	for addr := range d.pendingNodesDel {
		delete(d.curNodes, addr)
	}
	for a := range d.pendingNodesAdd {
		delete(d.pendingNodesAdd, a)
	}
	for a := range d.pendingNodesDel {
		delete(d.pendingNodesDel, a)
	}
	return nil
}

func (d *detailsState) CacheDetailsAbort() error {
	for a := range d.pendingNodesAdd {
		delete(d.pendingNodesAdd, a)
	}
	for a := range d.pendingNodesDel {
		delete(d.pendingNodesDel, a)
	}

	return nil
}
