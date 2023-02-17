// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/binary"

	"github.com/ava-labs/avalanchego/database"
)

func PrefixUserTypesKey(id uint64) (k []byte) {
	k = make([]byte, 10)
	k[0] = usersPrefix
	k[1] = ByteDelimiter

	binary.BigEndian.PutUint64(k[2:], id)

	return
}

func baseUserTypesPrefix() (k []byte) {
	k = make([]byte, 2)
	k[0] = userTypesPrefix
	k[1] = ByteDelimiter
	return
}

var _ UserTypesState = &userTypesState{}

type UserType struct {
	TypeName string `serialize:"true" json:"typeName"`
	TypeID   uint64 `serialize:"true" json:"typeID"`
	FeeUnits uint64 `serialize:"true" json:"fee"`
}

type UserTypesState interface {
	GetUserType(db database.Database, id uint64) (*UserType, bool, error)
	AddUserType(db database.Database, pmeta *UserType) error
	DelUserType(db database.Database, id uint64) error
	GetUserTypes(db database.Database) (map[uint64]*UserType, error)
	ReloadUserTypes(db database.Database) error
	CacheUserTypesCommit() error
	CacheUserTypesAbort() error

	CheckUserType(id uint64) bool
	CheckFee(id uint64, fee uint64) bool
}

type userTypesState struct {
	curUserTypes map[uint64]*UserType

	pendingUserTypesAdd map[uint64]*UserType
	pendingUserTypesDel map[uint64]*UserType
}

func NewUserTypesState(db database.Database) (*userTypesState, error) {
	s := &userTypesState{
		curUserTypes:        make(map[uint64]*UserType),
		pendingUserTypesAdd: make(map[uint64]*UserType),
		pendingUserTypesDel: make(map[uint64]*UserType),
	}
	err := s.ReloadUserTypes(db)
	return s, err
}

func (s *userTypesState) GetUserType(db database.Database, id uint64) (*UserType, bool, error) {
	pmeta, ok := s.curUserTypes[id]
	if !ok {
		return nil, false, nil
	}
	return pmeta, true, nil

}

func (s *userTypesState) CheckUserType(id uint64) bool {
	_, ok := s.curUserTypes[id]
	return ok
}

func (s *userTypesState) CheckFee(id uint64, fee uint64) bool {
	userType, ok := s.curUserTypes[id]
	if !ok {
		return ok
	}
	if fee == userType.FeeUnits {
		return true
	}
	return false
}

func (s *userTypesState) AddUserType(db database.Database, pmeta *UserType) error {
	id := pmeta.TypeID

	delete(s.pendingUserTypesDel, id)
	s.pendingUserTypesAdd[id] = pmeta

	k := PrefixUserTypesKey(id)
	pvmeta, err := Marshal(pmeta)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (s *userTypesState) DelUserType(db database.Database, id uint64) error {
	delete(s.pendingUserTypesAdd, id)
	s.pendingUserTypesDel[id] = &UserType{}

	k := PrefixUserTypesKey(id)
	return db.Delete(k)
}

func (s *userTypesState) GetUserTypes(db database.Database) (map[uint64]*UserType, error) {
	return s.curUserTypes, nil
}

func (s *userTypesState) ReloadUserTypes(db database.Database) error {
	basePrefix := baseUserTypesPrefix()
	cursor := db.NewIteratorWithPrefix(basePrefix)
	defer cursor.Release()
	for cursor.Next() {
		nmeta := cursor.Value()

		pmeta := new(UserType)
		if _, err := Unmarshal(nmeta, pmeta); err != nil {
			return err
		}
		s.curUserTypes[pmeta.TypeID] = pmeta

	}
	return nil
}

func (s *userTypesState) CacheUserTypesCommit() error {
	for id, pendingUserType := range s.pendingUserTypesAdd {
		delete(s.curUserTypes, id)
		s.curUserTypes[id] = pendingUserType
	}
	for id := range s.pendingUserTypesDel {
		delete(s.curUserTypes, id)
	}

	for a := range s.pendingUserTypesAdd {
		delete(s.pendingUserTypesAdd, a)
	}
	for a := range s.pendingUserTypesDel {
		delete(s.pendingUserTypesDel, a)
	}
	return nil
}

func (s *userTypesState) CacheUserTypesAbort() error {
	for a := range s.pendingUserTypesAdd {
		delete(s.pendingUserTypesAdd, a)
	}
	for a := range s.pendingUserTypesDel {
		delete(s.pendingUserTypesDel, a)
	}
	return nil
}
