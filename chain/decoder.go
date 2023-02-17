// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/SamaNetwork/SamaVM/tdata"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	Set      = "set"
	Transfer = "transfer"
	Stake    = "stake"
	UnStake  = "unstake"

	AddUser = "addUser"

	Govern    = "govern"
	Proof     = "proof"
	Claim     = "claim"
	Register  = "register"
	Refresh   = "refresh"
	Withdrawn = "withdrawn"
	Vote      = "vote"
	Proposal  = "proposal"
)

type Input struct {
	Typ          string         `json:"type"`
	Key          string         `json:"key"`
	Value        []byte         `json:"value"`
	To           common.Address `json:"to"`
	Units        uint64         `json:"units"`
	StakerType   uint64         `json:"stakerType"`
	StakerAddr   common.Address `json:"stakerAddr"`
	StakeAmount  uint64         `json:"stakeAmount"`
	RewardAmount uint64         `json:"rewardAmount"`
	ActionID     ids.ShortID    `json:"actionID"`
	LocalIP      string         `json:"localIP"`
	MinPort      uint64         `json:"minPort"`
	MaxPort      uint64         `json:"maxPort"`
	PublicIP     string         `json:"publicIP"`
	CheckPort    uint64         `json:"checkPort"`
	Netflow      uint64         `json:"netflow"`
	StartTime    uint64         `json:"startTime"`
	EndTime      uint64         `json:"endTime"`
	Ser          common.Address `json:"ser"`
	NewValue     string         `json:"newValue"`
	Country      string         `json:"country"`
	WorkKey      string         `json:"workKey"`
}

func (i *Input) Decode() (UnsignedTransaction, error) {
	switch i.Typ {
	case Set:
		return &SetTx{
			BaseTx: &BaseTx{},
			Value:  i.Value,
		}, nil
	case Transfer:
		return &TransferTx{
			BaseTx: &BaseTx{},
			To:     i.To,
			Units:  i.Units,
		}, nil
	case Stake:
		return &StakeTx{
			BaseTx:      &BaseTx{},
			StakerType:  i.StakerType,
			StakeAmount: i.StakeAmount,
			StakerAddr:  i.StakerAddr,
		}, nil
	case UnStake:
		return &UnStakeTx{
			BaseTx:       &BaseTx{},
			StakerType:   i.StakerType,
			RewardAmount: i.RewardAmount,
		}, nil
	case Register:
		return &RegisterTx{
			BaseTx:     &BaseTx{},
			ActionID:   i.ActionID,
			StakerType: i.StakerType,
			Country:    i.Country,
			WorkKey:    i.WorkKey,
			LocalIP:    i.LocalIP,
			MinPort:    i.MinPort,
			MaxPort:    i.MaxPort,
			PublicIP:   i.PublicIP,
			CheckPort:  i.CheckPort,
			StakerAddr: i.StakerAddr,
		}, nil
	case Refresh:
		return &RefreshTx{
			BaseTx:    &BaseTx{},
			Country:   i.Country,
			WorkKey:   i.WorkKey,
			LocalIP:   i.LocalIP,
			MinPort:   i.MinPort,
			MaxPort:   i.MaxPort,
			PublicIP:  i.PublicIP,
			CheckPort: i.CheckPort,
		}, nil
	case Proof:
		return &ProofTx{
			BaseTx:    &BaseTx{},
			Netflow:   i.Netflow,
			StartTime: i.StartTime,
			EndTime:   i.EndTime,
			Ser:       i.Ser,
		}, nil
	case Govern:
		return &GovernTx{
			BaseTx:   &BaseTx{},
			ActionID: i.ActionID,
		}, nil
	case Claim:
		return &ClaimTx{
			BaseTx:       &BaseTx{},
			RewardAmount: i.RewardAmount,
			EndTime:      i.EndTime,
		}, nil
	case Vote:
		return &VoteTx{
			ActionID: i.ActionID,
		}, nil
	case Proposal:
		return &ProposalTx{
			BaseTx:    &BaseTx{},
			ActionID:  i.ActionID,
			Key:       i.Key,
			NewValue:  i.NewValue,
			StartTime: i.StartTime,
			EndTime:   i.EndTime,
		}, nil
	default:
		return nil, ErrInvalidType
	}
}

const (
	tdString  = "string"
	tdUint64  = "uint64"
	tdBytes   = "bytes"
	tdAddress = "address"

	tdBlockID = "blockID"
	tdPrice   = "price"

	tdValue = "value"
	tdUnits = "units"
	tdTo    = "to"

	tdStakerType = "stakerType"
	tdStakeAddr  = "stakerAddr"
	tdAmount     = "stakeAmount"

	tdEndTime     = "endTime"
	tdStartTime   = "startTime"
	tdConnections = "connections"
	tdActionID    = "actionID"
	tdKey         = "key"

	tdReward  = "reward"
	tdSer     = "ser"
	tdNetflow = "netflow"

	tdLocalIP   = "localIP"
	tdMinPort   = "minPort"
	tdMaxPort   = "maxPort"
	tdPublicIP  = "publicIP"
	tdCheckPort = "checkPort"
	tdNewValue  = "newValue"

	tdCountry = "country"
	tdDevID   = "devID"
	tdWorkKey = "workKey"

	tdUserType   = "userType"
	tdActionType = "actionType"
)

func parseUint64Message(td *tdata.TypedData, k string) (uint64, error) {
	r, ok := td.Message[k].(string)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, k)
	}
	return strconv.ParseUint(r, 10, 64)
}

func parseBaseTx(td *tdata.TypedData) (*BaseTx, error) {
	rblockID, ok := td.Message[tdBlockID].(string)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdBlockID)
	}
	blockID, err := ids.FromString(rblockID)
	if err != nil {
		return nil, err
	}
	magic, err := strconv.ParseUint(td.Domain.Magic, 10, 64)
	if err != nil {
		return nil, err
	}
	price, err := parseUint64Message(td, tdPrice)
	if err != nil {
		return nil, err
	}
	return &BaseTx{BlockID: blockID, Magic: magic, Price: price}, nil
}

func ParseTypedData(td *tdata.TypedData) (UnsignedTransaction, error) {
	bTx, err := parseBaseTx(td)
	if err != nil {
		return nil, err
	}

	switch td.PrimaryType {
	case Set:
		rvalue, ok := td.Message[tdValue].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdValue)
		}
		value, err := hexutil.Decode(rvalue)
		if err != nil {
			return nil, err
		}
		return &SetTx{BaseTx: bTx, Value: value}, nil
	case Transfer:
		to, ok := td.Message[tdTo].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdTo)
		}
		units, err := parseUint64Message(td, tdUnits)
		if err != nil {
			return nil, err
		}
		return &TransferTx{BaseTx: bTx, To: common.HexToAddress(to), Units: units}, nil
	case Stake:
		stakerAddr, ok := td.Message[tdStakeAddr].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdStakeAddr)
		}

		stakerType, err := parseUint64Message(td, tdStakerType)
		if err != nil {
			return nil, err
		}

		stakeAmount, err := parseUint64Message(td, tdAmount)
		if err != nil {
			return nil, err
		}
		return &StakeTx{BaseTx: bTx, StakerType: stakerType, StakeAmount: stakeAmount, StakerAddr: common.HexToAddress(stakerAddr)}, nil
	case UnStake:
		stakerType, err := parseUint64Message(td, tdStakerType)
		if err != nil {
			return nil, err
		}

		rewardAmount, err := parseUint64Message(td, tdAmount)
		if err != nil {
			return nil, err
		}

		endTime, err := parseUint64Message(td, tdEndTime)
		if err != nil {
			return nil, err
		}
		return &UnStakeTx{BaseTx: bTx, StakerType: stakerType, RewardAmount: rewardAmount, EndTime: endTime}, nil
	case Register:
		ractionID, ok := td.Message[tdActionID].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdActionID)
		}
		actionID, _ := ids.ShortFromString(ractionID)

		localIP, ok := td.Message[tdLocalIP].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdLocalIP)
		}

		country, ok := td.Message[tdCountry].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdCountry)
		}

		workKey, ok := td.Message[tdWorkKey].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdWorkKey)
		}

		minPort, err := parseUint64Message(td, tdMinPort)
		if err != nil {
			return nil, err
		}
		maxPort, err := parseUint64Message(td, tdMaxPort)
		if err != nil {
			return nil, err
		}

		publicIP, ok := td.Message[tdPublicIP].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdPublicIP)
		}

		checkPort, err := parseUint64Message(td, tdCheckPort)
		if err != nil {
			return nil, err
		}

		stakerType, err := parseUint64Message(td, tdStakerType)
		if err != nil {
			return nil, err
		}

		stakerAddr, ok := td.Message[tdStakeAddr].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdStakeAddr)
		}

		return &RegisterTx{BaseTx: bTx, ActionID: actionID, StakerType: stakerType, Country: country, WorkKey: workKey, LocalIP: localIP, MinPort: minPort, MaxPort: maxPort,
			PublicIP: publicIP, CheckPort: checkPort, StakerAddr: common.HexToAddress(stakerAddr)}, nil
	case Refresh:
		localIP, ok := td.Message[tdLocalIP].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdLocalIP)
		}
		country, ok := td.Message[tdCountry].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdCountry)
		}

		workKey, ok := td.Message[tdWorkKey].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdWorkKey)
		}

		minPort, err := parseUint64Message(td, tdMinPort)
		if err != nil {
			return nil, err
		}
		maxPort, err := parseUint64Message(td, tdMaxPort)
		if err != nil {
			return nil, err
		}
		publicIP, ok := td.Message[tdPublicIP].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdPublicIP)
		}
		checkPort, err := parseUint64Message(td, tdCheckPort)
		if err != nil {
			return nil, err
		}
		return &RefreshTx{BaseTx: bTx, Country: country, WorkKey: workKey, LocalIP: localIP, MinPort: minPort, MaxPort: maxPort,
			PublicIP: publicIP, CheckPort: checkPort}, nil
	case Proof:
		netflow, err := parseUint64Message(td, tdNetflow)
		if err != nil {
			return nil, err
		}
		startTime, err := parseUint64Message(td, tdStartTime)
		if err != nil {
			return nil, err
		}
		endTime, err := parseUint64Message(td, tdEndTime)
		if err != nil {
			return nil, err
		}
		ser, ok := td.Message[tdSer].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdSer)
		}
		return &ProofTx{BaseTx: bTx, Netflow: netflow, StartTime: startTime,
			EndTime: endTime, Ser: common.HexToAddress(ser)}, nil
	case Govern:
		ractionID, ok := td.Message[tdActionID].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdActionID)
		}

		actionID, _ := ids.ShortFromString(ractionID)

		return &GovernTx{BaseTx: bTx, ActionID: actionID}, nil
	case Proposal:
		ractionID, ok := td.Message[tdActionID].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdActionID)
		}

		actionID, _ := ids.ShortFromString(ractionID)
		key, ok := td.Message[tdKey].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdTo)
		}

		value, ok := td.Message[tdNewValue].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdNewValue)
		}

		startTime, err := parseUint64Message(td, tdStartTime)
		if err != nil {
			return nil, err
		}
		endTime, err := parseUint64Message(td, tdEndTime)
		if err != nil {
			return nil, err
		}
		return &ProposalTx{BaseTx: bTx, ActionID: actionID, Key: key, NewValue: value, StartTime: startTime, EndTime: endTime}, nil
	case Claim:
		rewardAmount, err := parseUint64Message(td, tdAmount)
		if err != nil {
			return nil, err
		}

		endTime, err := parseUint64Message(td, tdEndTime)
		if err != nil {
			return nil, err
		}
		return &ClaimTx{BaseTx: bTx, RewardAmount: rewardAmount, EndTime: endTime}, nil
	case Vote:
		ractionID, ok := td.Message[tdActionID].(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrTypedDataKeyMissing, tdActionID)
		}

		actionID, _ := ids.ShortFromString(ractionID)
		return &VoteTx{BaseTx: bTx, ActionID: actionID}, nil
	default:
		return nil, ErrInvalidType
	}
}
