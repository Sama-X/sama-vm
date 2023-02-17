// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/inconshreveable/log15"
)

const (
	stakerTypeRoute     = routePrefix
	stakerTypeSer       = serPrefix
	stakerTypeValidator = validatorPrefix
)

func RouteStake() uint64 {
	return stakerTypeRoute
}

func SerStake() uint64 {
	return stakerTypeSer
}

func ValidatorStake() uint64 {
	return stakerTypeValidator
}

func ActionTypeModifySysParam() uint64 {
	return actionTypeModifySysParam
}

func ActionTypeModifyFoundation() uint64 {
	return actionTypeModifyFoundation
}

func ActionTypeAddStaker() uint64 {
	return actionTypeAddStaker
}

type StakerMeta struct {
	TxID        ids.ID         `serialize:"true" json:"txId"`
	StakerType  uint64         `serialize:"true" json:"stakerType"`
	StakeAmount uint64         `serialize:"true" json:"stakeAmount"`
	StakeTime   uint64         `serialize:"true" json:"stakeTime"`
	StakerAddr  common.Address `serialize:"true" json:"stakerAddr"`
}

func PrefixStaker4Key(stakerType byte, address common.Address) (k []byte) {
	k = make([]byte, 4+common.AddressLength)
	k[0] = stakerPrefix
	k[1] = ByteDelimiter
	k[2] = stakerType
	k[3] = ByteDelimiter
	copy(k[4:], address[:])
	return k
}

func baseStakerPrefix(stakerType byte) (k []byte) {
	k = make([]byte, 4)
	k[0] = stakerPrefix
	k[1] = ByteDelimiter
	k[2] = stakerType
	k[3] = ByteDelimiter
	return
}

var _ StakerState = &stakerState{}
var StakerTypes = []byte{stakerTypeRoute, stakerTypeSer, stakerTypeValidator}

type StakerState interface {
	GetPendingStakers(stakerType byte) (map[common.Address]*StakerMeta, map[common.Address]*StakerMeta, error)
	GetStakerMeta(stakerType byte, address common.Address) (*StakerMeta, bool, error)
	PutStaker(db database.Database, pmeta *StakerMeta) error
	GetStakers(stakerType byte) ([]*StakerMeta, error)
	DelStaker(db database.Database, stakerType byte, address common.Address) error
	ReloadStakers(db database.Database) error
	CacheStakersCommit() error
	CacheStakersAbort() error
	GetStakersNum() (int, int, int)

	IsRoute(address common.Address) (bool, error)
	IsSer(address common.Address) (bool, error)
	IsValidator(address common.Address) (bool, error)

	IsStaker(address common.Address) (bool, byte, error)
}

type stakerState struct {
	curRoute     map[common.Address]*StakerMeta
	curSer       map[common.Address]*StakerMeta
	curValidator map[common.Address]*StakerMeta

	pendingRouteAdd     map[common.Address]*StakerMeta
	pendingSerAdd       map[common.Address]*StakerMeta
	pendingValidatorAdd map[common.Address]*StakerMeta

	pendingRouteDel     map[common.Address]*StakerMeta
	pendingSerDel       map[common.Address]*StakerMeta
	pendingValidatorDel map[common.Address]*StakerMeta
}

func NewStakerState(db database.Database) (*stakerState, error) {
	state := &stakerState{
		curRoute:     make(map[common.Address]*StakerMeta),
		curSer:       make(map[common.Address]*StakerMeta),
		curValidator: make(map[common.Address]*StakerMeta),

		pendingRouteAdd:     make(map[common.Address]*StakerMeta),
		pendingSerAdd:       make(map[common.Address]*StakerMeta),
		pendingValidatorAdd: make(map[common.Address]*StakerMeta),

		pendingRouteDel:     make(map[common.Address]*StakerMeta),
		pendingSerDel:       make(map[common.Address]*StakerMeta),
		pendingValidatorDel: make(map[common.Address]*StakerMeta),
	}
	err := state.ReloadStakers(db)
	return state, err
}

func (s *stakerState) GetPendingStakers(stakerType byte) (map[common.Address]*StakerMeta, map[common.Address]*StakerMeta, error) {
	switch stakerType {
	case stakerTypeSer:
		return s.pendingSerAdd, s.pendingSerDel, nil
	case stakerTypeRoute:
		return s.pendingRouteAdd, s.pendingRouteDel, nil
	case stakerTypeValidator:
		return s.pendingValidatorAdd, s.pendingValidatorDel, nil
	default:
		return nil, nil, ErrIDErr
	}
}

func (s *stakerState) GetCurStakes(stakerType byte) (map[common.Address]*StakerMeta, error) {
	switch stakerType {
	case stakerTypeSer:
		return s.curSer, nil
	case stakerTypeRoute:
		return s.curRoute, nil
	case stakerTypeValidator:
		return s.curValidator, nil
	default:
		return nil, ErrIDErr
	}
}

func (s *stakerState) GetStakersNum() (int, int, int) {
	return len(s.curRoute), len(s.curSer), len(s.curValidator)
}

func (s *stakerState) IsStaker(address common.Address) (bool, byte, error) {
	ok := false
	for _, stakerType := range StakerTypes {
		curMap, _ := s.GetCurStakes(stakerType)
		_, ok = curMap[address]
		if ok {
			return ok, stakerType, nil
		}
	}

	return ok, 0, nil
}

func (s *stakerState) IsRoute(address common.Address) (bool, error) {
	curMap, _ := s.GetCurStakes(stakerTypeRoute)
	_, ok := curMap[address]
	return ok, nil
}

func (s *stakerState) IsSer(address common.Address) (bool, error) {
	curMap, _ := s.GetCurStakes(stakerTypeSer)
	_, ok := curMap[address]
	return ok, nil
}

func (s *stakerState) IsValidator(address common.Address) (bool, error) {
	curMap, _ := s.GetCurStakes(stakerTypeValidator)
	_, ok := curMap[address]
	return ok, nil
}

func (s *stakerState) GetStakerMeta(stakerType byte, address common.Address) (*StakerMeta, bool, error) {
	curMap, _ := s.GetCurStakes(stakerType)
	pmeta, ok := curMap[address]
	if !ok {
		return nil, false, nil
	}
	return pmeta, true, nil
}

func (s *stakerState) PutStaker(db database.Database, staker *StakerMeta) error {
	pendingAdd, pendingDel, _ := s.GetPendingStakers(byte(staker.StakerType))
	delete(pendingDel, staker.StakerAddr)
	pendingAdd[staker.StakerAddr] = staker

	k := PrefixStaker4Key(byte(staker.StakerType), staker.StakerAddr)
	pvmeta, err := Marshal(staker)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (s *stakerState) GetStakers(stakerType byte) ([]*StakerMeta, error) {
	stakers := []*StakerMeta(nil)
	curMap, _ := s.GetCurStakes(stakerType)
	for _, staker := range curMap {
		stakers = append(stakers, staker)
	}
	return stakers, nil
}

func (s *stakerState) DelStaker(db database.Database, stakerType byte, address common.Address) error {
	pendingAdd, pendingDel, _ := s.GetPendingStakers(stakerType)
	delete(pendingAdd, address)
	pendingDel[address] = &StakerMeta{}

	k := PrefixStaker4Key(stakerType, address)
	return db.Delete(k)
}

func (s *stakerState) ReloadStakers(db database.Database) error {
	for _, stakerType := range StakerTypes {
		log.Debug("ReloadStakers start", "stakerType", stakerType)
		basePrefix := baseStakerPrefix(stakerType)
		cursor := db.NewIteratorWithPrefix(basePrefix)
		defer cursor.Release()
		for cursor.Next() {
			log.Debug("ReloadStakers", "stakerType", stakerType)
			nmeta := cursor.Value()

			pmeta := new(StakerMeta)
			if _, err := Unmarshal(nmeta, pmeta); err != nil {
				return err
			}
			curMap, _ := s.GetCurStakes(stakerType)
			curMap[pmeta.StakerAddr] = pmeta
		}
	}
	return nil
}

func (s *stakerState) CacheStakersCommit() error {
	for _, stakerType := range StakerTypes {
		pendingAdd, pendingDel, _ := s.GetPendingStakers(stakerType)
		curMap, _ := s.GetCurStakes(stakerType)

		for addr, pendingReward := range pendingAdd {
			delete(curMap, addr)
			curMap[addr] = pendingReward
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

func (s *stakerState) CacheStakersAbort() error {
	for _, stakerType := range StakerTypes {
		pendingAdd, pendingDel, _ := s.GetPendingStakers(stakerType)
		for a := range pendingAdd {
			delete(pendingAdd, a)
		}
		for a := range pendingDel {
			delete(pendingDel, a)
		}
	}
	return nil
}
