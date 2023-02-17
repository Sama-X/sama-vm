// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import "github.com/ava-labs/avalanchego/ids"

type Activity struct {
	Tmstmp       int64  `serialize:"true" json:"timestamp"`
	TxID         ids.ID `serialize:"true" json:"txId"`
	Typ          string `serialize:"true" json:"type"`
	Sender       string `serialize:"true" json:"sender,omitempty"`
	Key          string `serialize:"true" json:"key,omitempty"`
	To           string `serialize:"true" json:"to,omitempty"` // common.Address will be 0x000 when not populated
	Units        uint64 `serialize:"true" json:"units,omitempty"`
	StakerType   uint64 `serialize:"true" json:"stakerType,omitempty"`
	StakeAmount  uint64 `serialize:"true" json:"stakeAmount,omitempty"`
	StakerAddr   string `serialize:"true" json:"stakerAddr,omitempty"`
	RewardAmount uint64 `serialize:"true" json:"rewardAmount,omitempty"`

	StartTime   uint64      `serialize:"true" json:"startTime,omitempty"`
	EndTime     uint64      `serialize:"true" json:"endTime,omitempty"`
	PayAmount   uint64      `serialize:"true" json:"payAmout,omitempty"`
	Connections uint64      `serialize:"true" json:"connections,omitempty"`
	Address     string      `serialize:"true" json:"address,omitempty"`
	ActionID    ids.ShortID `serialize:"true" json:"actionID,omitempty"`
	ParamType   uint64      `serialize:"true" json:"paramType,omitempty"`
	Value       string      `serialize:"true" json:"value,omitempty"`

	Netflow uint64 `serialize:"true" json:"netflow,omitempty"`

	LocalIP   string `serialize:"true" json:"localIP,omitempty"`
	MinPort   uint64 `serialize:"true" json:"minPort,omitempty"`
	MaxPort   uint64 `serialize:"true" json:"maxPort,omitempty"`
	PublicIP  string `serialize:"true" json:"publicIP,omitempty"`
	CheckPort uint64 `serialize:"true" json:"checkPort,omitempty"`
	Country   string `serialize:"true" json:"country,omitempty"`
	//DevID     string `serialize:"true" json:"devID,omitempty"`
	WorkKey    string `serialize:"true" json:"workKey,omitempty"`
	UserType   uint64 `serialize:"true" json:"userType,omitempty"`
	ActionType uint64 `serialize:"true" json:"actionType,omitempty"`
}
