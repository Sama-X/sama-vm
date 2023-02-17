// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/cache/metercacher"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	UsersCacheSize = 8096
)

func PrefixUsersKey(address common.Address) (k []byte) {
	k = make([]byte, 2+common.AddressLength)
	k[0] = usersPrefix
	k[1] = ByteDelimiter
	copy(k[2:], address[:])
	return
}

func baseUsersPrefix() (k []byte) {
	k = make([]byte, 2)
	k[0] = usersPrefix
	k[1] = ByteDelimiter
	return
}

var _ UsersState = &usersState{}

type UserMeta struct {
	StartTime   uint64         `serialize:"true" json:"startTime"`
	EndTime     uint64         `serialize:"true" json:"endTime"`
	Connections uint64         `serialize:"true" json:"connections"`
	PayAmount   uint64         `serialize:"true" json:"payAmount"`
	TxsID       []ids.ID       `serialize:"true" json:"txsid"`
	UserType    uint64         `serialize:"true" json:"userType"`
	Address     common.Address `serialize:"true" json:"address"`
}

type UsersState interface {
	GetUserMeta(db database.Database, address common.Address) (*UserMeta, bool, error)
	PutUser(db database.Database, pmeta *UserMeta) error
	DelUser(db database.Database, address common.Address) error
	GetUsers(db database.Database) ([]*UserMeta, error)
	ReloadUsers(db database.Database) error
	CacheUsersCommit() error
	CacheUsersAbort() error
}

type usersState struct {
	curUsers cache.Cacher[common.Address, *UserMeta]

	pendingUsersAdd map[common.Address]*UserMeta
	pendingUsersDel map[common.Address]*UserMeta
}

func NewUserstate(db database.Database, metrics prometheus.Registerer) (*usersState, error) {
	curUsers, err := metercacher.New[common.Address, *UserMeta](
		"users_cache",
		metrics,
		&cache.LRU[common.Address, *UserMeta]{Size: UsersCacheSize},
	)
	if err != nil {
		return nil, err
	}
	s := &usersState{
		curUsers:        curUsers,
		pendingUsersAdd: make(map[common.Address]*UserMeta),
		pendingUsersDel: make(map[common.Address]*UserMeta),
	}
	err = s.ReloadUsers(db)
	return s, err
}

func (s *usersState) GetUserMeta(db database.Database, address common.Address) (*UserMeta, bool, error) {
	if award, found := s.curUsers.Get(address); found {
		if award == nil {
			return nil, false, database.ErrNotFound
		}
		return award, true, nil
	}
	k := PrefixUsersKey(address)
	cmeta, err := db.Get(k)
	if err != nil {
		if err != database.ErrNotFound {
			return nil, false, err
		}
		return nil, false, nil
	}
	pmeta := new(UserMeta)
	if _, err := Unmarshal(cmeta, pmeta); err != nil {
		return nil, false, err
	}

	return pmeta, true, nil
}

func (s *usersState) PutUser(db database.Database, pmeta *UserMeta) error {
	address := pmeta.Address

	delete(s.pendingUsersDel, address)
	s.pendingUsersAdd[address] = pmeta

	k := PrefixUsersKey(address)
	pvmeta, err := Marshal(pmeta)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (s *usersState) DelUser(db database.Database, address common.Address) error {
	delete(s.pendingUsersAdd, address)
	s.pendingUsersDel[address] = &UserMeta{}

	k := PrefixUsersKey(address)
	return db.Delete(k)
}

func (s *usersState) GetUsers(db database.Database) ([]*UserMeta, error) {
	users := []*UserMeta{}
	basePrefix := baseUsersPrefix()
	cursor := db.NewIteratorWithPrefix(basePrefix)
	defer cursor.Release()
	for cursor.Next() {
		nmeta := cursor.Value()

		pmeta := new(UserMeta)
		if _, err := Unmarshal(nmeta, pmeta); err != nil {
			return nil, err
		}
		users = append(users, pmeta)
	}
	return users, nil
}

func (s *usersState) ReloadUsers(db database.Database) error {
	count := 0
	basePrefix := baseUsersPrefix()
	cursor := db.NewIteratorWithPrefix(basePrefix)
	defer cursor.Release()
	for cursor.Next() {
		nmeta := cursor.Value()

		pmeta := new(UserMeta)
		if _, err := Unmarshal(nmeta, pmeta); err != nil {
			return err
		}
		s.curUsers.Put(pmeta.Address, pmeta)
		count++
		if count > UsersCacheSize/2 {
			return nil
		}
	}
	return nil
}

func (s *usersState) CacheUsersCommit() error {
	for addr, pendingUser := range s.pendingUsersAdd {
		s.curUsers.Evict(addr)
		s.curUsers.Put(addr, pendingUser)
	}
	for addr := range s.pendingUsersDel {
		s.curUsers.Evict(addr)
	}
	for a := range s.pendingUsersAdd {
		delete(s.pendingUsersAdd, a)
	}
	for a := range s.pendingUsersDel {
		delete(s.pendingUsersDel, a)
	}
	return nil
}

func (s *usersState) CacheUsersAbort() error {
	for a := range s.pendingUsersAdd {
		delete(s.pendingUsersAdd, a)
	}
	for a := range s.pendingUsersDel {
		delete(s.pendingUsersDel, a)
	}
	return nil
}
