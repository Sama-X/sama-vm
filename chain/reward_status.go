// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

const (
	LocalPrefix  = 0
	GlobalPrefix = 1
)

func PrefixRewardKey(address common.Address) (k []byte) {
	k = make([]byte, 4+common.AddressLength)
	k[0] = rewardPrefix
	k[1] = ByteDelimiter
	k[2] = LocalPrefix
	k[3] = ByteDelimiter
	copy(k[4:], address[:])
	return
}

func baseRewardPrefix() (k []byte) {
	k = make([]byte, 4)
	k[0] = rewardPrefix
	k[1] = ByteDelimiter
	k[2] = LocalPrefix
	k[3] = ByteDelimiter

	return
}

func RewardGlobalPrefix() (k []byte) {
	k = make([]byte, 4)
	k[0] = rewardPrefix
	k[1] = ByteDelimiter
	k[2] = GlobalPrefix
	k[3] = ByteDelimiter

	return
}

var _ RewardState = &rewardState{}

type RewardGlobal struct {
	LastOprTime uint64 `serialize:"true" json:"lastOprTime"`
	LastOprTXID ids.ID `serialize:"true" json:"lastOprTxId"`
}

type RewardMeta struct {
	YieldReward   uint64         `serialize:"true" json:"yieldReward"`
	BaseReward    uint64         `serialize:"true" json:"BaseReward"`
	MeritReward   uint64         `serialize:"true" json:"meritReward"`
	LastOprTime   uint64         `serialize:"true" json:"lastOprTime"`
	LastOprTXID   ids.ID         `serialize:"true" json:"lastOprTxId"`
	LastClaimTime uint64         `serialize:"true" json:"lastClaimTime"`
	LastClaimTXID ids.ID         `serialize:"true" json:"lastClaimTxId"`
	RewardAddr    common.Address `serialize:"true" json:"rewardAddr"`
}

type RewardState interface {
	GetRewardMeta(address common.Address) (*RewardMeta, bool, error)
	UpdateReward(db database.Database, pG *RewardGlobal, pmeta *RewardMeta) error
	UpdateOwner(db database.Database, pmeta *RewardMeta) error
	UpdateGlobal(db database.Database, pG *RewardGlobal) error
	GetRewards() ([]*RewardMeta, error)
	DelReward(db database.Database, address common.Address) error
	ReloadRewards(db database.Database) error
	CacheRewardsCommit() error
	CacheRewardsAbort() error
	GetLastUpdateTime() (uint64, error)
	GetLastClaimTime(address common.Address) (uint64, error)
}

type rewardState struct {
	curGlobal         *RewardGlobal
	pendigGlobal      *RewardGlobal
	curRewards        map[common.Address]*RewardMeta
	pendingRewardsAdd map[common.Address]*RewardMeta
	pendingRewardsDel map[common.Address]*RewardMeta
}

func NewRewardState(db database.Database) (*rewardState, error) {
	state := &rewardState{
		curGlobal: &RewardGlobal{
			LastOprTime: 0,
			LastOprTXID: ids.ID{},
		},
		pendigGlobal: &RewardGlobal{
			LastOprTime: 0,
			LastOprTXID: ids.ID{},
		},
		curRewards:        make(map[common.Address]*RewardMeta),
		pendingRewardsAdd: make(map[common.Address]*RewardMeta),
		pendingRewardsDel: make(map[common.Address]*RewardMeta),
	}
	err := state.ReloadRewards(db)
	return state, err
}

func (r *rewardState) GetLastUpdateTime() (uint64, error) {
	if r.pendigGlobal.LastOprTime > r.curGlobal.LastOprTime {
		return r.pendigGlobal.LastOprTime, nil
	}
	return r.curGlobal.LastOprTime, nil
}

func (r *rewardState) GetLastClaimTime(address common.Address) (uint64, error) {
	pmeta, ok := r.curRewards[address]
	if !ok {
		return 0, nil
	}
	return pmeta.LastClaimTime, nil
}

func (r *rewardState) GetRewardMeta(address common.Address) (*RewardMeta, bool, error) {
	pmeta, ok := r.curRewards[address]
	if !ok {
		return nil, false, nil
	}
	return pmeta, true, nil
}

func (r *rewardState) UpdateReward(db database.Database, pG *RewardGlobal, pmeta *RewardMeta) error {
	address := pmeta.RewardAddr

	delete(r.pendingRewardsDel, address)
	r.pendingRewardsAdd[address] = pmeta
	k := PrefixRewardKey(address)
	pvmeta, err := Marshal(pmeta)
	if err != nil {
		return err
	}
	err = db.Put(k, pvmeta)
	if err != nil {
		return err
	}
	k = RewardGlobalPrefix()
	pvG, err := Marshal(pG)
	if err != nil {
		return err
	}
	return db.Put(k, pvG)

}

func (r *rewardState) UpdateOwner(db database.Database, pmeta *RewardMeta) error {
	address := pmeta.RewardAddr

	delete(r.pendingRewardsDel, address)
	r.pendingRewardsAdd[address] = pmeta
	k := PrefixRewardKey(address)
	pvmeta, err := Marshal(pmeta)
	if err != nil {
		return err
	}
	err = db.Put(k, pvmeta)
	return err
}

func (r *rewardState) UpdateGlobal(db database.Database, pG *RewardGlobal) error {
	k := RewardGlobalPrefix()
	pvG, err := Marshal(pG)
	if err != nil {
		return err
	}
	return db.Put(k, pvG)
}

func (r *rewardState) GetRewards() ([]*RewardMeta, error) {
	rewards := []*RewardMeta(nil)
	for _, reward := range r.curRewards {
		rewards = append(rewards, reward)
	}
	return rewards, nil
}

func (r *rewardState) DelReward(db database.Database, address common.Address) error {
	delete(r.pendingRewardsAdd, address)
	r.pendingRewardsDel[address] = &RewardMeta{}

	k := PrefixRewardKey(address)
	return db.Delete(k)
}

func (r *rewardState) ReloadRewards(db database.Database) error {
	basePrefix := baseRewardPrefix()
	cursor := db.NewIteratorWithPrefix(basePrefix)
	defer cursor.Release()
	for cursor.Next() {
		nmeta := cursor.Value()

		pmeta := new(RewardMeta)
		if _, err := Unmarshal(nmeta, pmeta); err != nil {
			return err
		}
		r.curRewards[pmeta.RewardAddr] = pmeta
	}

	k := RewardGlobalPrefix()
	ymeta, err := db.Get(k)
	if err != nil {
		if err == database.ErrNotFound {
			return nil
		}
		return err
	}
	if _, err := Unmarshal(ymeta, r.curGlobal); err != nil {
		return err
	}

	return nil
}

func (r *rewardState) CacheRewardsCommit() error {
	for addr, pendingReward := range r.pendingRewardsAdd {
		delete(r.curRewards, addr)
		r.curRewards[addr] = pendingReward
	}
	for addr := range r.pendingRewardsDel {
		delete(r.curRewards, addr)
	}
	for a := range r.pendingRewardsAdd {
		delete(r.pendingRewardsAdd, a)
	}
	for a := range r.pendingRewardsDel {
		delete(r.pendingRewardsDel, a)
	}

	if r.pendigGlobal.LastOprTime != 0 {
		*r.curGlobal = *r.pendigGlobal
		r.pendigGlobal.LastOprTime = 0
		r.pendigGlobal.LastOprTXID = ids.ID{}
	}
	return nil
}

func (r *rewardState) CacheRewardsAbort() error {
	for a := range r.pendingRewardsAdd {
		delete(r.pendingRewardsAdd, a)
	}
	for a := range r.pendingRewardsDel {
		delete(r.pendingRewardsDel, a)
	}
	r.pendigGlobal.LastOprTime = 0
	r.pendigGlobal.LastOprTXID = ids.ID{}

	return nil
}
