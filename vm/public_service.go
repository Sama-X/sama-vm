// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	log "github.com/inconshreveable/log15"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/SamaNetwork/SamaVM/tdata"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type PublicService struct {
	vm    *VM
	Cache map[string]*LocalParams
}

type PingReply struct {
	Success bool `serialize:"true" json:"success"`
}

func (svc *PublicService) Ping(_ *http.Request, _ *struct{}, reply *PingReply) (err error) {
	log.Info("ping")
	reply.Success = true
	return nil
}

type NetworkReply struct {
	NetworkID uint32 `serialize:"true" json:"networkId"`
	SubnetID  ids.ID `serialize:"true" json:"subnetId"`
	ChainID   ids.ID `serialize:"true" json:"chainId"`
}

func (svc *PublicService) Network(_ *http.Request, _ *struct{}, reply *NetworkReply) (err error) {
	reply.NetworkID = svc.vm.ctx.NetworkID
	reply.SubnetID = svc.vm.ctx.SubnetID
	reply.ChainID = svc.vm.ctx.ChainID
	return nil
}

type GenesisReply struct {
	Genesis *chain.Genesis `serialize:"true" json:"genesis"`
}

func (svc *PublicService) Genesis(_ *http.Request, _ *struct{}, reply *GenesisReply) (err error) {
	reply.Genesis = svc.vm.Genesis()
	return nil
}

type IssueRawTxArgs struct {
	Tx []byte `serialize:"true" json:"tx"`
}

type IssueRawTxReply struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

func (svc *PublicService) IssueRawTx(_ *http.Request, args *IssueRawTxArgs, reply *IssueRawTxReply) error {
	tx := new(chain.Transaction)
	if _, err := chain.Unmarshal(args.Tx, tx); err != nil {
		return err
	}

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(svc.vm.genesis); err != nil {
		return err
	}
	reply.TxID = tx.ID()

	errs := svc.vm.Submit(nil, tx)
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("%v", errs)
}

type IssueTxArgs struct {
	TypedData *tdata.TypedData `serialize:"true" json:"typedData"`
	Signature hexutil.Bytes    `serialize:"true" json:"signature"`
}

type IssueTxReply struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

func (svc *PublicService) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	if args.TypedData == nil {
		return ErrTypedDataIsNil
	}
	utx, err := chain.ParseTypedData(args.TypedData)
	if err != nil {
		return err
	}
	tx := chain.NewTx(utx, args.Signature[:])

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(svc.vm.genesis); err != nil {
		return err
	}
	reply.TxID = tx.ID()

	errs := svc.vm.Submit(nil, tx)
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("%v", errs)
}

type HasTxArgs struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

type HasTxReply struct {
	Accepted bool `serialize:"true" json:"accepted"`
}

func (svc *PublicService) HasTx(_ *http.Request, args *HasTxArgs, reply *HasTxReply) error {
	has, err := chain.HasTransaction(svc.vm.db, args.TxID)
	if err != nil {
		return err
	}
	reply.Accepted = has
	return nil
}

type LastAcceptedReply struct {
	Height  uint64 `serialize:"true" json:"height"`
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

func (svc *PublicService) LastAccepted(_ *http.Request, _ *struct{}, reply *LastAcceptedReply) error {
	la := svc.vm.lastAccepted
	reply.Height = la.Hght
	reply.BlockID = la.ID()
	return nil
}

type SuggestedFeeArgs struct {
	Input *chain.Input `serialize:"true" json:"input"`
}

type SuggestedFeeReply struct {
	TypedData *tdata.TypedData `serialize:"true" json:"typedData"`
	TotalCost uint64           `serialize:"true" json:"totalCost"`
}

func (svc *PublicService) SuggestedFee(
	_ *http.Request,
	args *SuggestedFeeArgs,
	reply *SuggestedFeeReply,
) error {
	if args.Input == nil {
		return ErrInputIsNil
	}
	utx, err := args.Input.Decode()
	if err != nil {
		return err
	}

	// Determine suggested fee
	price, cost, err := svc.vm.SuggestedFee()
	if err != nil {
		return err
	}
	g := svc.vm.genesis
	fu := utx.FeeUnits(g)
	if fu != 0 {
		price += cost / fu
	}

	// Update meta
	utx.SetBlockID(svc.vm.lastAccepted.ID())
	utx.SetMagic(g.Magic)
	utx.SetPrice(price)

	reply.TypedData = utx.TypedData()
	reply.TotalCost = fu * price
	return nil
}

type SuggestedRawFeeReply struct {
	Price uint64 `serialize:"true" json:"price"`
	Cost  uint64 `serialize:"true" json:"cost"`
}

func (svc *PublicService) SuggestedRawFee(
	_ *http.Request,
	_ *struct{},
	reply *SuggestedRawFeeReply,
) error {
	price, cost, err := svc.vm.SuggestedFee()
	if err != nil {
		return err
	}
	reply.Price = price
	reply.Cost = cost
	return nil
}

type ResolveArgs struct {
	Key common.Hash `serialize:"true" json:"key"`
}

type ResolveReply struct {
	Exists    bool             `serialize:"true" json:"exists"`
	Value     []byte           `serialize:"true" json:"value"`
	ValueMeta *chain.ValueMeta `serialize:"true" json:"valueMeta"`
}

func (svc *PublicService) Resolve(_ *http.Request, args *ResolveArgs, reply *ResolveReply) error {
	vmeta, exists, err := chain.GetValueMeta(svc.vm.db, args.Key)
	if err != nil {
		return err
	}
	if !exists {
		// Avoid value lookup if doesn't exist
		return nil
	}
	v, exists, err := chain.GetValue(svc.vm.db, args.Key)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCorruption
	}

	// Set values properly
	reply.Exists = true
	reply.Value = v
	reply.ValueMeta = vmeta
	return nil
}

type BalanceArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type BalanceReply struct {
	Balance uint64 `serialize:"true" json:"balance"`
}

func (svc *PublicService) Balance(_ *http.Request, args *BalanceArgs, reply *BalanceReply) error {
	bal, err := chain.GetBalance(svc.vm.db, args.Address)
	if err != nil {
		return err
	}
	reply.Balance = bal
	return err
}

type RecentActivityReply struct {
	Activity []*chain.Activity `serialize:"true" json:"activity"`
}

func (svc *PublicService) RecentActivity(_ *http.Request, _ *struct{}, reply *RecentActivityReply) error {
	cs := uint64(svc.vm.config.ActivityCacheSize)
	if cs == 0 {
		return nil
	}

	// Sort results from newest to oldest
	start := svc.vm.activityCacheCursor
	i := start
	activity := []*chain.Activity{}
	for i > 0 && start-i < cs {
		i--
		item := svc.vm.activityCache[i%cs]
		if item == nil {
			break
		}
		activity = append(activity, item)
	}
	reply.Activity = activity
	return nil
}

type StakeBalanceArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type StakeBalanceReply struct {
	Balance uint64 `serialize:"true" json:"amount"`
}

func (svc *PublicService) StakeBalance(_ *http.Request, args *StakeBalanceArgs, reply *StakeBalanceReply) error {
	bal, err := chain.GetStakeBalance(svc.vm.db, args.Address)
	if err != nil {
		return err
	}
	reply.Balance = bal
	return err
}

type APIStake struct {
	TxID        ids.ID         `serialize:"true" json:"txId"`
	StakerType  string         `serialize:"true" json:"stakerType"`
	StakerAddr  common.Address `serialize:"true" json:"stakerAddr"`
	StakeAmount uint64         `serialize:"true" json:"stakeAmount"`
	StakeTime   uint64         `serialize:"true" json:"stakeTime"`
}

type GetStakersArgs struct {
	StakerType uint64         `serialize:"true" json:"stakerType"`
	Address    common.Address `serialize:"true" json:"address"`
}

type GetStakersReply struct {
	Stakers []APIStake `serialize:"true"  json:"stakes"`
}

var zeroAddress = (common.Address{})

func (svc *PublicService) GetStakers(_ *http.Request, args *GetStakersArgs, reply *GetStakersReply) error {
	if (args.StakerType != chain.RouteStake()) && (args.StakerType != chain.SerStake()) {
		return fmt.Errorf("type err %d", args.StakerType)
	}

	if bytes.Equal(args.Address[:], zeroAddress[:]) {
		stakers, err := svc.vm.samaState.GetStakers(byte(args.StakerType))
		if err != nil {
			return fmt.Errorf("couldn't GetStakers %w", err)
		}

		for _, staker := range stakers {
			strType := "normal"
			if staker.StakerType == chain.RouteStake() {
				strType = "Route"
			} else if staker.StakerType == chain.SerStake() {
				strType = "Ser"
			}

			reply.Stakers = append(reply.Stakers, APIStake{
				TxID:        staker.TxID,
				StakerType:  strType,
				StakerAddr:  staker.StakerAddr,
				StakeAmount: staker.StakeAmount,
				StakeTime:   staker.StakeTime,
			})
		}

	} else {
		staker, exist, err := svc.vm.samaState.GetStakerMeta(byte(args.StakerType), args.Address)
		if err != nil {
			return fmt.Errorf("getStakeMeta error %w", err)
		}
		if !exist {
			return fmt.Errorf("not found")
		}

		strType := "normal"
		if staker.StakerType == chain.RouteStake() {
			strType = "Route"
		} else if staker.StakerType == chain.SerStake() {
			strType = "Ser"
		}

		reply.Stakers = append(reply.Stakers, APIStake{
			TxID:        staker.TxID,
			StakerType:  strType,
			StakerAddr:  staker.StakerAddr,
			StakeAmount: staker.StakeAmount,
			StakeTime:   staker.StakeTime,
		})

	}

	return nil
}

type CalcRewardArgs struct {
	StakerType uint64         `serialize:"true" json:"stakerType"`
	Address    common.Address `serialize:"true" json:"address"`
	EndTime    uint64         `serialize:"true" json:"endTime"`
}

type CalcRewardReply struct {
	Yield uint64 `serialize:"true" json:"yield"`
	Base  uint64 `serialize:"true" json:"base"`
	Merit uint64 `serialize:"true" json:"merit"`
}

func (svc *PublicService) CalcReward(_ *http.Request, args *CalcRewardArgs, reply *CalcRewardReply) error {
	//if time.Now().After(time.Unix(int64(args.EndTime+chain.EffectiveSecs), 0)) {
	//	return fmt.Errorf("too early %d-%d ", time.Now().Unix(), args.EndTime)
	//}

	if time.Now().Before(time.Unix(int64(args.EndTime), 0)) {
		return fmt.Errorf("too late %d-%d ", time.Now().Unix(), args.EndTime)
	}
	base, merit, yield, err := svc.vm.samaState.CalcReward(byte(args.StakerType), args.Address, args.EndTime)
	if err != nil {
		return fmt.Errorf("calc reward error %w", err)
	}
	reply.Yield = yield
	reply.Base = base
	reply.Merit = merit

	return err
}

func (svc *PublicService) SignSubmitRawTx(
	ctx context.Context,
	utx chain.UnsignedTransaction,
	priv *ecdsa.PrivateKey,
) (txID ids.ID, cost uint64, err error) {

	price, cost, err := svc.vm.SuggestedFee()
	if err != nil {
		return ids.Empty, 0, err
	}
	g := svc.vm.genesis
	fu := utx.FeeUnits(g)
	if fu != 0 {
		price += cost / fu
	}

	la := svc.vm.lastAccepted.ID()

	blockCost := fu * price

	utx.SetBlockID(la)
	utx.SetMagic(g.Magic)
	if utx.FeeUnits(g) != 0 {
		utx.SetPrice(price + blockCost/utx.FeeUnits(g))
	} else {
		utx.SetPrice(price)
	}

	dh, err := chain.DigestHash(utx)
	if err != nil {
		return ids.Empty, 0, err
	}

	sig, err := chain.Sign(dh, priv)
	if err != nil {
		return ids.Empty, 0, err
	}
	tx := chain.NewTx(utx, sig)
	if err := tx.Init(g); err != nil {
		return ids.Empty, 0, err
	}

	errs := svc.vm.Submit(nil, tx)
	if len(errs) == 0 {
		return tx.ID(), blockCost, nil
	}
	if len(errs) == 1 {
		return tx.ID(), blockCost, errs[0]
	}
	return tx.ID(), blockCost, fmt.Errorf("%v", errs)
}

type LocalParams struct {
	NetParams
	WorkKey     string         `serialize:"true" json:"workKey"`
	WorkAddress common.Address `serialize:"true" json:"workAddress"`
}

type NetParams struct {
	Country   string `serialize:"true" json:"country"`
	LocalIP   string `serialize:"true" json:"localIP"`
	PublicIP  string `serialize:"true" json:"publicIP"`
	MinPort   uint64 `serialize:"true" json:"minPort"`
	MaxPort   uint64 `serialize:"true" json:"maxPort"`
	CheckPort uint64 `serialize:"true" json:"checkPort"`
	//DevID     string `serialize:"true" json:"devID"`
}

type RegisterArgs struct {
	LocalParams
}

type RegisterReply struct {
	Succ bool `serialize:"true" json:"bSucc"`
}

func (svc *PublicService) Register(_ *http.Request, args *RegisterArgs, reply *RegisterReply) error {

	pbkb, err := hex.DecodeString(args.WorkKey)
	if err != nil {
		return err
	}
	pbk, err := ethcrypto.UnmarshalPubkey(pbkb) //(*ecdsa.PublicKey, error)
	if err != nil {
		return err
	}
	addr := ethcrypto.PubkeyToAddress(*pbk)

	params := new(LocalParams)
	*params = args.LocalParams
	params.WorkAddress = addr

	svc.Cache[params.WorkAddress.String()] = params
	reply.Succ = true
	return nil
}

type GetLocalParamsArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type GetLocalParamsReply struct {
	Params *LocalParams `serialize:"true" json:"params"`
}

func (svc *PublicService) GetLocalParams(_ *http.Request, args *GetLocalParamsArgs, reply *GetLocalParamsReply) error {
	if bytes.Equal(args.Address[:], zeroAddress[:]) {
		return fmt.Errorf("err Missing Address")
	}

	params, exist := svc.Cache[args.Address.String()]
	if !exist {
		return fmt.Errorf("not found %s", args.Address.String())
	}
	reply.Params = params
	return nil
}

// ImportKeyReply is the response for ImportKey
type ImportKeyReply struct {
	// The address controlled by the PrivateKey provided in the arguments
	Address common.Address `json:"address"`
}

// ImportKeyArgs are arguments for ImportKey
type ImportKeyArgs struct {
	api.UserPass
	PrivateKey *crypto.PrivateKeySECP256K1R `json:"privateKey"`
}

// ImportKey adds a private key to the provided user
func (svc *PublicService) ImportKey(r *http.Request, args *ImportKeyArgs, reply *ImportKeyReply) error {
	svc.vm.ctx.Log.Debug("sama: ImportKey called",
		logging.UserString("username", args.Username),
	)

	if args.PrivateKey == nil {
		return fmt.Errorf("err Missing PrivateKey ")
	}

	db, err := svc.vm.ctx.Keystore.GetDatabase(args.Username, args.Password)
	if err != nil {
		return fmt.Errorf("%s %s problem retrieving data: %w ", args.Username, args.Password, err)
	}
	defer db.Close()
	pubKey := args.PrivateKey.PublicKey().(*crypto.PublicKeySECP256K1R)
	reply.Address = ethcrypto.PubkeyToAddress(*(pubKey.ToECDSA()))
	user := userKey{
		db: db,
	}
	if err := user.putAddress(args.PrivateKey); err != nil {
		return fmt.Errorf("problem saving key %w", err)
	}

	return nil
}

// ExportKeyArgs are arguments for ExportKey
type ExportKeyArgs struct {
	api.UserPass
	Address string `json:"address"`
}

// ExportKeyReply is the response for ExportKey
type ExportKeyReply struct {
	// The decrypted PrivateKey for the Address provided in the arguments
	PrivateKey    *crypto.PrivateKeySECP256K1R `json:"privateKey"`
	PrivateKeyHex string                       `json:"privateKeyHex"`
}

func ParseEthAddress(addrStr string) (common.Address, error) {
	if !common.IsHexAddress(addrStr) {
		return common.Address{}, fmt.Errorf("invalid Address")
	}
	return common.HexToAddress(addrStr), nil
}

// ExportKey returns a private key from the provided user
func (svc *PublicService) ExportKey(r *http.Request, args *ExportKeyArgs, reply *ExportKeyReply) error {
	svc.vm.ctx.Log.Debug("sama: ExportKey called",
		logging.UserString("username", args.Username),
	)

	address, err := ParseEthAddress(args.Address)
	if err != nil {
		return fmt.Errorf("couldn't parse %s to address", args.Address)
	}

	db, err := svc.vm.ctx.Keystore.GetDatabase(args.Username, args.Password)
	if err != nil {
		return fmt.Errorf("problem retrieving user '%s': %w", args.Username, err)
	}
	defer db.Close()

	user := userKey{
		db: db,
	}
	reply.PrivateKey, err = user.getKey(address)
	if err != nil {
		return fmt.Errorf("problem retrieving private key: %w", err)
	}
	reply.PrivateKeyHex = hexutil.Encode(reply.PrivateKey.Bytes())

	return nil
}

type GetNodesArgs struct {
	Address     common.Address `serialize:"true" json:"workAddr"`
	IsConfirmed bool           `serialize:"true" json:"isConfirmed"`
}

type APINode struct {
	TxID       ids.ID         `serialize:"true" json:"txId"`
	StakerType string         `serialize:"true" json:"stakerType"`
	StakerAddr common.Address `serialize:"true" json:"stakerAddr"`
	Country    string         `serialize:"true" json:"country"`
	WorkKey    string         `serialize:"true" json:"workKey"`
	LocalIP    string         `serialize:"true" json:"localIP"`
	MinPort    uint64         `serialize:"true" json:"minPort"`
	MaxPort    uint64         `serialize:"true" json:"maxPort"`
	PublicIP   string         `serialize:"true" json:"publicIP"`
	CheckPort  uint64         `serialize:"true" json:"checkPort"`
	WorkAddr   common.Address `serialize:"true" json:"workAddr"`
}

type GetNodesReply struct {
	Nodes []APINode `serialize:"true"  json:"nodes"`
}

func (svc *PublicService) GetNodes(_ *http.Request, args *GetNodesArgs, reply *GetNodesReply) error {

	if bytes.Equal(args.Address[:], zeroAddress[:]) {
		nodes, err := svc.vm.samaState.GetDetails()
		if err != nil {
			return fmt.Errorf("couldn't GetStakers %w", err)
		}
		for _, node := range nodes {
			strType := "normal"
			if node.StakerType == chain.RouteStake() {
				if args.IsConfirmed {
					ok, err := svc.vm.samaState.IsRoute(node.StakeAddress)
					if err != nil {
						return err
					}
					if !ok {
						continue
					}
				}
				strType = "Route"
			} else if node.StakerType == chain.SerStake() {
				if args.IsConfirmed {
					ok, err := svc.vm.samaState.IsSer(node.StakeAddress)
					if err != nil {
						return err
					}
					if !ok {
						continue
					}
				}
				strType = "Ser"
			}

			reply.Nodes = append(reply.Nodes, APINode{
				TxID:       node.TxID,
				StakerType: strType,
				StakerAddr: node.StakeAddress,
				WorkAddr:   node.WorkAddress,
				Country:    node.Country,
				WorkKey:    node.WorkKey,
				LocalIP:    node.LocalIP,
				MinPort:    node.MinPort,
				MaxPort:    node.MaxPort,
				PublicIP:   node.PublicIP,
				CheckPort:  node.CheckPort,
			})
		}
	} else {
		node, exist, err := svc.vm.samaState.GetDetailMeta(args.Address)
		if err != nil {
			return fmt.Errorf("getStakeMeta error %w", err)
		}
		if !exist {
			return fmt.Errorf("not found")
		}
		strType := "normal"
		if node.StakerType == chain.RouteStake() {
			if args.IsConfirmed {
				ok, err := svc.vm.samaState.IsRoute(node.StakeAddress)
				if err != nil {
					return err
				}
				if !ok {
					return err
				}
			}
			strType = "Route"
		} else if node.StakerType == chain.SerStake() {
			if args.IsConfirmed {
				ok, err := svc.vm.samaState.IsSer(node.StakeAddress)
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
			strType = "Ser"
		}

		reply.Nodes = append(reply.Nodes, APINode{
			TxID:       node.TxID,
			StakerType: strType,
			StakerAddr: node.StakeAddress,
			WorkAddr:   node.WorkAddress,
			Country:    node.Country,
			WorkKey:    node.WorkKey,
			LocalIP:    node.LocalIP,
			MinPort:    node.MinPort,
			MaxPort:    node.MaxPort,
			PublicIP:   node.PublicIP,
			CheckPort:  node.CheckPort,
		})
	}
	return nil
}

type GetActionsArgs struct {
	ActionType uint64      `serialize:"true" json:"actionType"`
	ActionID   ids.ShortID `serialize:"true" json:"actionID"`
}

type APIAction struct {
	TxID       ids.ID           `serialize:"true" json:"txId"`
	ActionID   ids.ShortID      `serialize:"true" json:"actionID"`
	ActionType uint64           `serialize:"true" json:"actionType"`
	StartTime  uint64           `serialize:"true" json:"startTime"`
	EndTime    uint64           `serialize:"true" json:"endTime"`
	Key        string           `serialize:"true" json:"key"`
	NewValue   string           `serialize:"true" json:"newValue"`
	Voters     []common.Address `serialize:"true" json:"voters"`
}

type GetActionsReply struct {
	Actions []APIAction `serialize:"true"  json:"actions"`
}

var zeroShortID = (ids.ShortID{})

func (svc *PublicService) GetActions(_ *http.Request, args *GetActionsArgs, reply *GetActionsReply) error {

	ok := svc.vm.samaState.CheckActionType(args.ActionType)
	if !ok {
		return fmt.Errorf("action type err")
	}

	if bytes.Equal(args.ActionID[:], zeroShortID[:]) {
		actions, err := svc.vm.samaState.GetActions()
		if err != nil {
			return fmt.Errorf("couldn't GetActions %w", err)
		}
		for _, action := range actions {
			reply.Actions = append(reply.Actions, APIAction{
				TxID:       action.TxIDs[0],
				ActionType: action.ActionType,
				ActionID:   action.ActionID,
				StartTime:  action.StartTime,
				EndTime:    action.EndTime,
				Key:        action.Key,
				NewValue:   action.NewValue,
				Voters:     action.Voters,
			})
		}
	} else {
		action, exist, err := svc.vm.samaState.GetActionMeta(args.ActionID)
		if err != nil {
			return fmt.Errorf("GetActionMeta error %w", err)
		}
		if !exist {
			return fmt.Errorf("not found")
		}
		reply.Actions = append(reply.Actions, APIAction{
			TxID:       action.TxIDs[0],
			ActionType: action.ActionType,
			ActionID:   action.ActionID,
			StartTime:  action.StartTime,
			EndTime:    action.EndTime,
			Key:        action.Key,
			NewValue:   action.NewValue,
			Voters:     action.Voters,
		})
	}
	return nil
}

type UserFeeArgs struct {
	UserType  uint64 `serialize:"true" json:"userType"`
	StartTime uint64 `serialize:"true" json:"startTime"`
	EndTime   uint64 `serialize:"true" json:"endTime"`
}

type UserFeeReply struct {
	PayAmount uint64 `serialize:"true" json:"payAmount"`
}

func (svc *PublicService) GetUserFee(_ *http.Request, args *UserFeeArgs, reply *UserFeeReply) (err error) {
	user, ok, err := svc.vm.samaState.GetUserType(svc.vm.db, args.UserType)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user type not found")
	}
	reply.PayAmount = user.FeeUnits

	return nil
}

type GetUsersArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type APIUser struct {
	TxID      ids.ID         `serialize:"true" json:"txId"`
	StartTime uint64         `serialize:"true" json:"startTime"`
	EndTime   uint64         `serialize:"true" json:"endTime"`
	Address   common.Address `serialize:"true" json:"address"`
}

type GetUsersReply struct {
	Users []APIUser `serialize:"true"  json:"users"`
}

func (svc *PublicService) GetUsers(_ *http.Request, args *GetUsersArgs, reply *GetUsersReply) error {
	if bytes.Equal(args.Address[:], zeroAddress[:]) {
		users, err := svc.vm.samaState.GetUsers(svc.vm.db)
		if err != nil {
			return fmt.Errorf("couldn't GetUsers %w", err)
		}
		for _, user := range users {
			reply.Users = append(reply.Users, APIUser{
				TxID:      user.TxsID[0],
				StartTime: user.StartTime,
				EndTime:   user.EndTime,
				Address:   user.Address,
			})
		}
	} else {
		user, exist, err := svc.vm.samaState.GetUserMeta(svc.vm.db, args.Address)
		if err != nil {
			return fmt.Errorf("GetUserMeta error %w", err)
		}
		if !exist {
			return fmt.Errorf("not found")
		}
		reply.Users = append(reply.Users, APIUser{
			TxID:      user.TxsID[0],
			StartTime: user.StartTime,
			EndTime:   user.EndTime,
			Address:   user.Address,
		})
	}
	return nil
}

type GetYieldsReply struct {
	Yields uint64 `serialize:"true"  json:"yields"`
}

func (svc *PublicService) GetChainYields(_ *http.Request, _ *struct{}, reply *GetYieldsReply) error {
	reply.Yields = svc.vm.samaState.GetChainYields()
	return nil
}

type GetStakerTypeArgs struct {
	Address common.Address `serialize:"true" json:"address"`
}

type GetStakerTypeReply struct {
	StakerType uint64 `serialize:"true"  json:"stakerType"`
}

func (svc *PublicService) GetStakerType(_ *http.Request, args *GetStakerTypeArgs, reply *GetStakerTypeReply) error {
	if bytes.Equal(args.Address[:], zeroAddress[:]) {
		return fmt.Errorf("address error")
	}
	ok, err := svc.vm.samaState.IsRoute(args.Address)
	if err != nil {
		return fmt.Errorf("IsRoute error %w", err)
	}
	if !ok {
		ok, err = svc.vm.samaState.IsRoute(args.Address)
		if err != nil {
			return fmt.Errorf("IsRoute error %w", err)
		}
		if !ok {
			return fmt.Errorf("not found")
		}
		reply.StakerType = chain.SerStake()
	} else {
		reply.StakerType = chain.RouteStake()
	}
	return nil
}

type GetCreateTimeReply struct {
	CreateTime uint64 `serialize:"true"  json:"createTime"`
}

func (svc *PublicService) GetChainCreateTime(_ *http.Request, _ *struct{}, reply *GetCreateTimeReply) error {
	reply.CreateTime = svc.vm.samaState.GetChainCreateTime()
	return nil
}

type GetPowsArgs struct {
	Type  uint64         `serialize:"true" json:"type"`
	Miner common.Address `serialize:"true" json:"miner"`
}

type APIPow struct {
	TotalTime      uint64         `serialize:"true" json:"workTime"`
	Totalflow      uint64         `serialize:"true" json:"totalflow"`
	LastUpdateTime uint64         `serialize:"true" json:"lastTime"`
	LastUpdateTXID ids.ID         `serialize:"true" json:"lastTxId"`
	Miner          common.Address `serialize:"true" json:"miner"`
}

type GetPowsReply struct {
	PowType string   `serialize:"true" json:"powType"`
	Pows    []APIPow `serialize:"true"  json:"pows"`
}

func (svc *PublicService) GetPows(_ *http.Request, args *GetPowsArgs, reply *GetPowsReply) error {
	if (args.Type != chain.RouteStake()) && (args.Type != chain.SerStake()) {
		return fmt.Errorf("type err %d", args.Type)
	}

	if bytes.Equal(args.Miner[:], zeroAddress[:]) {
		pows, err := svc.vm.samaState.GetPows(byte(args.Type))
		if err != nil {
			return fmt.Errorf("couldn't GetPows %w", err)
		}

		for _, pow := range pows {
			strType := "normal"
			if pow.PowType == chain.RouteStake() {
				strType = "Route"
			} else if pow.PowType == chain.SerStake() {
				strType = "Ser"
			}

			reply.PowType = strType
			reply.Pows = append(reply.Pows, APIPow{
				TotalTime:      pow.TotalTime,
				Totalflow:      pow.Totalflow,
				LastUpdateTime: pow.LastUpdateTime,
				LastUpdateTXID: pow.LastUpdateTXID,
				Miner:          pow.Miner,
			})
		}

	} else {
		pow, exist, err := svc.vm.samaState.GetPowMeta(byte(args.Type), args.Miner)
		if err != nil {
			return fmt.Errorf("GetPowMeta error %w", err)
		}
		if !exist {
			return fmt.Errorf("not found")
		}

		strType := "normal"
		if pow.PowType == chain.RouteStake() {
			strType = "Route"
		} else if pow.PowType == chain.SerStake() {
			strType = "Ser"
		}
		reply.PowType = strType
		reply.Pows = append(reply.Pows, APIPow{
			TotalTime:      pow.TotalTime,
			Totalflow:      pow.Totalflow,
			LastUpdateTime: pow.LastUpdateTime,
			LastUpdateTXID: pow.LastUpdateTXID,
			Miner:          pow.Miner,
		})
	}

	return nil
}

var letterBytes = []byte("1234567890abcdefghijklmnopqrstuvwxyz")

func RandStringRunes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))] //nolint:Replygosec
	}
	return b
}

type CreateShortReply struct {
	ShortID ids.ShortID `serialize:"true" json:"shortID"`
}

func (svc *PublicService) CreateShortID(_ *http.Request, _ *struct{}, reply *CreateShortReply) error {
	ub := RandStringRunes(20)
	shortID, err := ids.ToShortID(ub)
	if err != nil {
		return fmt.Errorf("couldn't parse to id: %w", err)
	}
	reply.ShortID = shortID
	return nil
}

type GetSysParamsReply struct {
	Params chain.SysParamsMeta `serialize:"true" json:"params"`
}

func (svc *PublicService) GetSysParams(_ *http.Request, _ *struct{}, reply *GetSysParamsReply) error {
	params := svc.vm.samaState.GetSysParams()
	reply.Params = *params
	return nil
}

type ProofArgs struct {
	api.UserPass
	Address   string `serialize:"true" json:"address"`
	Netflow   uint64 `serialize:"true" json:"netflow"`
	StartTime uint64 `serialize:"true" json:"startTime"`
	EndTime   uint64 `serialize:"true" json:"endTime"`
}

type ProofReply struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

func (svc *PublicService) Proof(_ *http.Request, args *ProofArgs, reply *ProofReply) error {
	address, err := ParseEthAddress(args.Address)
	if err != nil {
		return fmt.Errorf("couldn't parse %s to address", args.Address)
	}

	db, err := svc.vm.ctx.Keystore.GetDatabase(args.Username, args.Password)
	if err != nil {
		return fmt.Errorf("problem retrieving user '%s': %w", args.Username, err)
	}
	defer db.Close()

	user := userKey{
		db: db,
	}
	privKey, err := user.getKey(address)
	if err != nil {
		return fmt.Errorf("problem retrieving private key: %w", err)
	}

	utx := &chain.ProofTx{
		BaseTx:    &chain.BaseTx{},
		Netflow:   args.Netflow,
		StartTime: args.StartTime,
		EndTime:   args.EndTime,
	}
	txId, _, err := svc.SignSubmitRawTx(context.Background(), utx, privKey.ToECDSA())
	if err != nil {
		return err
	}

	reply.TxID = txId
	return nil
}

type RefreshArgs struct {
	api.UserPass
	Address   string `serialize:"true" json:"workAddr"`
	Country   string `serialize:"true" json:"country"`
	LocalIP   string `serialize:"true" json:"localIP"`
	PublicIP  string `serialize:"true" json:"publicIP"`
	MinPort   uint64 `serialize:"true" json:"minPort"`
	MaxPort   uint64 `serialize:"true" json:"maxPort"`
	CheckPort uint64 `serialize:"true" json:"checkPort"`
	WorkKey   string `serialize:"true" json:"workKey"`
}

type RefreshReply struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

func (svc *PublicService) Refresh(_ *http.Request, args *RefreshArgs, reply *RefreshReply) error {
	address, err := ParseEthAddress(args.Address)
	if err != nil {
		return fmt.Errorf("couldn't parse %s to address", args.Address)
	}

	db, err := svc.vm.ctx.Keystore.GetDatabase(args.Username, args.Password)
	if err != nil {
		return fmt.Errorf("problem retrieving user '%s': %w", args.Username, err)
	}
	defer db.Close()

	user := userKey{
		db: db,
	}
	privKey, err := user.getKey(address)
	if err != nil {
		return fmt.Errorf("problem retrieving private key: %w", err)
	}

	//node, exist, err := svc.vm.samaState.GetDetailMeta(address)
	//if err != nil {
	//	return err
	//}
	//if !exist {
	//	return fmt.Errorf("not found %s", address)
	//}
	localIP := args.LocalIP
	minPort := args.MinPort
	maxPort := args.MaxPort
	publicIP := args.PublicIP
	checkPort := args.CheckPort
	country := args.Country
	workKey := args.WorkKey

	utx := &chain.RefreshTx{
		BaseTx:    &chain.BaseTx{},
		Country:   country,
		WorkKey:   workKey,
		LocalIP:   localIP,
		MinPort:   minPort,
		MaxPort:   maxPort,
		PublicIP:  publicIP,
		CheckPort: checkPort,
	}
	txId, _, err := svc.SignSubmitRawTx(context.Background(), utx, privKey.ToECDSA())
	if err != nil {
		return err
	}

	reply.TxID = txId
	return nil
}
