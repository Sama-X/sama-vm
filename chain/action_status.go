// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
)

const (
	actionTypeStart = iota
	actionTypeAddStaker
	actionTypeModifySysParam
	actionTypeAddUserType
	actionTypeModifyUserType
	actionTypeModifyFoundation
	actionTypeEnd
)

func PrefixActionsKey(actionID ids.ShortID) (k []byte) {
	k = make([]byte, 2+len(actionID))
	k[0] = actionsPrefix
	k[1] = ByteDelimiter

	copy(k[2:], actionID[:])
	return
}

func baseActionsPrefix() (k []byte) {
	k = make([]byte, 2)
	k[0] = actionsPrefix
	k[1] = ByteDelimiter
	return
}

var _ ActionsState = &actionsState{}

type ActionMeta struct {
	ActionID   ids.ShortID      `serialize:"true" json:"actionID"`
	ActionType uint64           `serialize:"true" json:"actionType"`
	StartTime  uint64           `serialize:"true" json:"startTime"`
	EndTime    uint64           `serialize:"true" json:"endTime"`
	TxIDs      []ids.ID         `serialize:"true" json:"txids"`
	Voters     []common.Address `serialize:"true" json:"voters"`
	Key        string           `serialize:"true" json:"key"`
	NewValue   string           `serialize:"true" json:"newValue"`
}

type ActionsState interface {
	GetActionMeta(actionID ids.ShortID) (*ActionMeta, bool, error)
	PutAction(db database.Database, actionID ids.ShortID, pmeta *ActionMeta) error
	GetActions() ([]*ActionMeta, error)
	DelAction(db database.Database, actionID ids.ShortID) error
	ReloadActions(db database.Database) error
	CacheActionsCommit() error
	CacheActionsAbort() error
	GetVotersNum(actionType uint64, key string) (int, ids.ShortID, error)
	ProposalRepeat(actionID ids.ShortID, key string, newValue string) error
	CheckActionType(actionType uint64) bool
}

type actionsState struct {
	curActions        map[ids.ShortID]*ActionMeta
	pendingActionsAdd map[ids.ShortID]*ActionMeta
	pendingActionsDel map[ids.ShortID]*ActionMeta
}

func NewActionsState(db database.Database) (*actionsState, error) {
	s := &actionsState{
		curActions:        make(map[ids.ShortID]*ActionMeta),
		pendingActionsAdd: make(map[ids.ShortID]*ActionMeta),
		pendingActionsDel: make(map[ids.ShortID]*ActionMeta),
	}

	err := s.ReloadActions(db)
	return s, err
}

func (s *actionsState) GetActionMeta(actionID ids.ShortID) (*ActionMeta, bool, error) {
	pmeta, ok := s.curActions[actionID]
	if !ok {
		return nil, false, nil
	}
	if pmeta.EndTime < uint64(time.Now().Unix()) {
		return nil, false, fmt.Errorf("action is overdue ")
	}
	return pmeta, true, nil
}

func (s *actionsState) PutAction(db database.Database, actionID ids.ShortID, pmeta *ActionMeta) error {
	delete(s.pendingActionsDel, actionID)
	s.pendingActionsAdd[actionID] = pmeta
	k := PrefixActionsKey(actionID)
	pvmeta, err := Marshal(pmeta)
	if err != nil {
		return err
	}
	return db.Put(k, pvmeta)
}

func (s *actionsState) GetActions() ([]*ActionMeta, error) {
	actions := []*ActionMeta(nil)
	for _, action := range s.curActions {
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *actionsState) GetVotersNum(actionType uint64, key string) (int, ids.ShortID, error) {
	for _, action := range s.curActions {
		if action.ActionType == actionType && action.Key == key {
			num := len(action.Voters)
			return num, action.ActionID, nil
		}
	}
	return 0, ids.ShortID{}, fmt.Errorf("not found ")
}

func (s *actionsState) DelAction(db database.Database, actionID ids.ShortID) error {
	delete(s.pendingActionsAdd, actionID)
	s.pendingActionsDel[actionID] = &ActionMeta{}

	k := PrefixActionsKey(actionID)
	return db.Delete(k)

}

func (s *actionsState) ReloadActions(db database.Database) error {
	basePrefix := baseActionsPrefix()
	cursor := db.NewIteratorWithPrefix(basePrefix)
	defer cursor.Release()
	for cursor.Next() {
		nmeta := cursor.Value()

		pmeta := new(ActionMeta)
		if _, err := Unmarshal(nmeta, pmeta); err != nil {
			return err
		}
		if pmeta.EndTime < uint64(time.Now().Unix()) {
			//delete
			continue
		}

		s.curActions[pmeta.ActionID] = pmeta
	}
	return nil
}

func (s *actionsState) CacheActionsCommit() error {
	for actionID, pendingAction := range s.pendingActionsAdd {
		delete(s.curActions, actionID)
		s.curActions[actionID] = pendingAction
	}

	for actionID := range s.pendingActionsDel {
		delete(s.curActions, actionID)
	}
	for actionID := range s.pendingActionsAdd {
		delete(s.pendingActionsAdd, actionID)
	}
	for actionID := range s.pendingActionsDel {
		delete(s.pendingActionsDel, actionID)
	}

	return nil
}

func (s *actionsState) CacheActionsAbort() error {
	for actionID := range s.pendingActionsAdd {
		delete(s.pendingActionsAdd, actionID)
	}
	for actionID := range s.pendingActionsDel {
		delete(s.pendingActionsDel, actionID)
	}

	return nil
}

func (s *actionsState) ProposalRepeat(actionID ids.ShortID, key string, newValue string) error {
	_, exist := s.curActions[actionID]
	if exist {
		return fmt.Errorf("action exist curActions")
	}
	for _, action := range s.curActions {
		if action.Key == key {
			return fmt.Errorf("key exist curActions")
		}
	}
	_, exist = s.pendingActionsAdd[actionID]
	if exist {
		return fmt.Errorf("action id exist pending")
	}
	for _, action := range s.pendingActionsAdd {
		if action.Key == key {
			return fmt.Errorf("key exist pending")
		}
	}

	return nil
}

func (s *actionsState) CheckActionType(actionType uint64) bool {
	if actionType > actionTypeStart && actionType < actionTypeEnd {
		return true
	}
	return false
}
