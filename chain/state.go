// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"encoding/hex"
	"fmt"
	"math"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

var _ SamaState = &samaState{}

var (
	SecondsYear   = uint64(364 * 24 * 60 * 60)
	SecondsMonth  = uint64(30 * 24 * 60 * 60)
	SecondsDay    = uint64(24 * 60 * 60)
	Seconds7Day   = uint64(7 * 24 * 60 * 60)
	SecondsMinute = uint64(60)
	HoursYear     = uint64(364 * 24)
)

type SamaState interface {
	SysParams
	StakerState
	RewardState
	PowState
	YieldsState
	UsersState
	DetailsState
	ActionsState
	UserTypesState
	Commit() error
	Abort() error
	CalcReward(claimerType byte, address common.Address, endTime uint64) (uint64, uint64, uint64, error)
	CheckPayAmount(db database.Database, userType uint64, amount uint64, startTime uint64, endTime uint64) (bool, error)

	DealStakeTx(db database.Database, staker *StakerMeta) error
	DealUnStakeTx(db database.Database, stakerType byte, address common.Address, txID ids.ID, endTime uint64) error
	DealAddUserTx(db database.Database, txID ids.ID, blkTime uint64, user *UserMeta) error
	UpdateNodeParams(db database.Database, detail *DetailMeta) error

	UpdateStakerReward(db database.Database, stakerType byte, address common.Address, txID ids.ID, endTime uint64) error
	UpdateFoundationReward(db database.Database, address common.Address, txID ids.ID, endTime uint64) error

	IsBeConfirmed(actionType uint64, key string) (bool, ids.ShortID, error)
	ProposalStatus(actionID ids.ShortID) (bool, error)
	IsValidWorkAddress(address common.Address) (bool, byte, error)
	CheckClaimAddress(address common.Address) (bool, byte, error)
}

type samaState struct {
	SysParams
	StakerState
	RewardState
	PowState
	YieldsState
	UsersState
	DetailsState
	ActionsState
	UserTypesState
}

func SamaNew(db database.Database, metrics prometheus.Registerer, g *Genesis) (SamaState, error) {
	sysParams, err := NewSysParamsState(db, g)
	if err != nil {
		return nil, err
	}
	stakeState, err := NewStakerState(db)
	if err != nil {
		return nil, err
	}
	rewardState, err := NewRewardState(db)
	if err != nil {
		return nil, err
	}
	powState, err := NewPowState(db)
	if err != nil {
		return nil, err
	}
	yieldsState, err := NewYieldsState(db)
	if err != nil {
		return nil, err
	}
	usersState, err := NewUserstate(db, metrics)
	if err != nil {
		return nil, err
	}
	detailsState, err := NewDetailstate(db)
	if err != nil {
		return nil, err
	}
	actionsState, err := NewActionsState(db)
	if err != nil {
		return nil, err
	}
	userTypesState, err := NewUserTypesState(db)
	if err != nil {
		return nil, err
	}
	return &samaState{
		SysParams:      sysParams,
		StakerState:    stakeState,
		RewardState:    rewardState,
		PowState:       powState,
		YieldsState:    yieldsState,
		UsersState:     usersState,
		DetailsState:   detailsState,
		ActionsState:   actionsState,
		UserTypesState: userTypesState,
	}, err
}

func (s *samaState) IsValidWorkAddress(address common.Address) (bool, byte, error) {
	nodes, err := s.GetDetails()
	if err != nil {
		return false, 0, err
	}
	ok := bool(false)
	for _, node := range nodes {
		pbkb, err := hex.DecodeString(node.WorkKey)
		if err != nil {
			return false, 0, err
		}
		pbk, err := crypto.UnmarshalPubkey(pbkb) //(*ecdsa.PublicKey, error)
		if err != nil {
			return false, 0, err
		}
		addr := crypto.PubkeyToAddress(*pbk)
		if addr == address {
			if node.StakerType == stakerTypeRoute {
				ok, err = s.IsRoute(node.StakeAddress)
				if err != nil {
					return false, 0, err
				}
			} else if node.StakerType == stakerTypeSer {
				ok, err = s.IsSer(node.StakeAddress)
				if err != nil {
					return false, 0, err
				}
			} else {
				return false, 0, fmt.Errorf("type err")
			}
			return ok, byte(node.StakerType), err
		}
	}
	return false, 0, err
}

func (s *samaState) Commit() error {
	err := s.CacheRewardsCommit()
	if err != nil {
		return err
	}
	err = s.CacheStakersCommit()
	if err != nil {
		return err
	}
	err = s.CacheDetailsCommit()
	if err != nil {
		return err
	}
	err = s.CacheActionsCommit()
	if err != nil {
		return err
	}
	err = s.CacheUsersCommit()
	if err != nil {
		return err
	}
	err = s.CachePowsCommit()
	if err != nil {
		return err
	}
	err = s.CacheYieldsCommit()
	if err != nil {
		return err
	}
	err = s.CacheParamsCommit()
	if err != nil {
		return err
	}
	err = s.CacheUserTypesCommit()
	if err != nil {
		return err
	}
	return err
}

func (s *samaState) Abort() error {
	err := s.CacheRewardsAbort()
	if err != nil {
		return err
	}
	err = s.CacheActionsAbort()
	if err != nil {
		return err
	}
	err = s.CacheUsersAbort()
	if err != nil {
		return err
	}
	err = s.CacheDetailsAbort()
	if err != nil {
		return err
	}
	err = s.CachePowsAbort()
	if err != nil {
		return err
	}
	err = s.CacheParamsAbort()
	if err != nil {
		return err
	}
	err = s.CacheYieldsAbort()
	if err != nil {
		return err
	}
	err = s.CacheUserTypesAbort()
	if err != nil {
		return err
	}
	err = s.CacheStakersAbort()
	return err

}

func (s *samaState) RewardCurYear(index uint32) uint64 {
	total := uint64(0)
	totalYears := s.GetTotalYears()
	changeYears := s.GetRateSustainYears()
	totalTokens := s.GetChainTokenAmount()
	for i := uint32(0); i < totalYears/changeYears; i++ {
		total += uint64(math.Pow(0.8, float64(i)))
	}
	quota := float64(1)
	if index != 0 {
		quota = float64(math.Pow(0.8, float64(index)))
	}

	totalReward := float64(totalTokens) * quota / float64(total)
	return uint64(totalReward)
}

func (s *samaState) StakePercentage(stakerType byte) uint32 {
	switch stakerType {
	case stakerTypeRoute:
		return s.GetPercRoute()
	case stakerTypeSer:
		return s.GetPercSer()
	case stakerTypeValidator:
		return s.GetPercValidator()
	default:
		return 0
	}
}

func (s *samaState) StakersNum(stakerType byte) int {
	routes, sers, validators := s.GetStakersNum()
	switch stakerType {
	case stakerTypeRoute:
		return routes
	case stakerTypeSer:
		return sers
	case stakerTypeValidator:
		return validators
	}
	return 0
}

func (s *samaState) StakePowMinutes(stakerType byte, address common.Address) (uint64, error) {
	pow, exist, err := s.GetPowMeta(stakerType, address)
	if err != nil {
		return 0, err
	}
	if !exist {
		return 0, nil
	}
	return pow.TotalTime, nil
}

func (s *samaState) ChainTotalPowMinutes(stakerType byte) (uint64, error) {
	total, err := s.TotalPowTime(stakerType)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *samaState) StakerReword(stakerType byte, roleNum int, address common.Address,
	stakeTime uint64, endTime uint64) (uint64, uint64, uint64, error) {

	baseReward := uint64(0)
	meritReward := uint64(0)
	yieldReward := uint64(0)
	meritInc := uint64(0)

	percBase := uint32(0)
	percMerit := uint32(0)
	switch stakerType {
	case stakerTypeRoute:
		percBase = s.GetRoutePercBase()
		percMerit = s.GetRoutePercMerit()
	case stakerTypeSer:
		percBase = s.GetSerPercBase()
		percMerit = s.GetSerPercMerit()
	case stakerTypeValidator:
		percBase = 100
	}

	startTime := stakeTime

	reward, exist, err := s.GetRewardMeta(address)
	if err != nil {
		return 0, 0, 0, err
	}
	if exist {
		startTime = reward.LastOprTime
		baseReward += reward.BaseReward
		meritReward += reward.MeritReward
		yieldReward += reward.YieldReward
	}
	totalYields := s.GetChainYields()
	roleYields := totalYields * uint64(s.StakePercentage(stakerType)) / 100
	yieldReward += roleYields / uint64(roleNum)

	createTime := s.GetChainCreateTime()
	sYear := (startTime - createTime) / SecondsYear
	eYear := (endTime - createTime) / SecondsYear
	log.Error("CalcReward start ")
	for i := sYear; i < eYear+1; i++ {
		tmpStart := createTime + i*SecondsYear
		tmpEnd := createTime + (i+1)*SecondsYear

		totalRewardYear := s.RewardCurYear(uint32(i))
		roleTotalYear := totalRewardYear * uint64(s.StakePercentage(stakerType)) / 100

		workSecs := uint64(0)
		if tmpEnd < endTime {
			if tmpStart > startTime {
				workSecs = tmpEnd - tmpStart
			} else {
				workSecs = tmpEnd - startTime
			}
		} else {
			if tmpStart > startTime {
				workSecs = endTime - tmpStart
			} else {
				workSecs = endTime - startTime
			}
		}

		if stakerType == stakerTypeValidator {
			baseReward += (roleTotalYear / (uint64(roleNum) * SecondsYear)) * workSecs
			meritReward = 0
		} else {
			baseTotal := roleTotalYear * uint64(percBase) / 100
			meritTotal := roleTotalYear * uint64(percMerit) / 100

			meritInc += (meritTotal / SecondsYear) * workSecs
			baseReward += (baseTotal / (uint64(roleNum) * SecondsYear)) * workSecs
		}
		if stakerType != stakerTypeValidator {
			totalTime, _ := s.ChainTotalPowMinutes(stakerType)
			userTime, _ := s.StakePowMinutes(stakerType, address)
			if totalTime != 0 {
				meritReward += meritInc * userTime / totalTime
			}
		}
	}

	return baseReward, meritReward, yieldReward, nil
}

func (s *samaState) FoundationReword(address common.Address, endTime uint64) (uint64, error) {
	createTime := s.GetChainCreateTime()
	startTime := createTime
	reward, exist, err := s.GetRewardMeta(address)
	if err != nil {
		return 0, err
	}
	if exist {
		startTime = reward.LastClaimTime
	}
	if endTime < startTime {
		return 0, fmt.Errorf("end time err")
	}

	fReward := uint64(0)
	sYear := (startTime - createTime) / SecondsYear
	eYear := (endTime - createTime) / SecondsYear
	log.Error("calc FoundationReword start ")
	for i := sYear; i < eYear+1; i++ {
		tmpStart := createTime + i*SecondsYear
		tmpEnd := createTime + (i+1)*SecondsYear

		totalRewardYear := s.RewardCurYear(uint32(i))
		roleTotalYear := totalRewardYear * uint64(s.GetPercFoundation()) / 100

		workSecs := uint64(0)
		if tmpEnd < endTime {
			if tmpStart > startTime {
				workSecs = tmpEnd - tmpStart
			} else {
				workSecs = tmpEnd - startTime
			}
		} else {
			if tmpStart > startTime {
				workSecs = endTime - tmpStart
			} else {
				workSecs = endTime - startTime
			}
		}

		fReward += (roleTotalYear / SecondsYear) * workSecs

	}
	return 0, nil
}

func (s *samaState) UpdateFoundationReward(db database.Database, address common.Address, txID ids.ID, endTime uint64) error {
	lastTime, _ := s.GetLastUpdateTime()
	if lastTime > endTime {
		return ErrEndTimeTooEarly
	} else if lastTime == endTime {
		return nil
	}

	base := uint64(0)

	base, err := s.FoundationReword(address, endTime)
	if err != nil {
		return err
	}

	err = s.UpdateOwner(db,
		&RewardMeta{
			BaseReward:    base,
			MeritReward:   0,
			YieldReward:   0,
			LastOprTime:   endTime,
			LastOprTXID:   txID,
			LastClaimTime: endTime,
			LastClaimTXID: txID,
			RewardAddr:    address,
		})
	if err != nil {
		return err
	}
	return nil
}

func (s *samaState) UpdateStakerReward(db database.Database, stakerType byte, address common.Address, txID ids.ID, endTime uint64) error {
	lastTime, _ := s.GetLastUpdateTime()
	if lastTime > endTime {
		return ErrEndTimeTooEarly
	} else if lastTime == endTime {
		return nil
	}
	roleNum := s.StakersNum(stakerType)
	if roleNum == 0 {
		return nil
	}
	stakers, err := s.GetStakers(stakerType)
	if err != nil {
		return err
	}
	for _, staker := range stakers {
		if staker.StakeTime >= endTime {
			//return ErrEndTimeTooEarly
			continue
		}

		base := uint64(0)
		merit := uint64(0)
		yield := uint64(0)
		if address != staker.StakerAddr {
			base, merit, yield, err = s.StakerReword(byte(staker.StakerType), roleNum, staker.StakerAddr, staker.StakeTime, endTime)
			if err != nil {
				return err
			}
		}
		err = s.UpdateOwner(db,
			&RewardMeta{
				BaseReward:    base,
				MeritReward:   merit,
				YieldReward:   yield,
				LastOprTime:   endTime,
				LastOprTXID:   txID,
				LastClaimTime: endTime,
				LastClaimTXID: txID,
				RewardAddr:    staker.StakerAddr,
			})
		if err != nil {
			return err
		}
	}
	err = s.UpdateGlobal(db, &RewardGlobal{
		LastOprTime: endTime,
		LastOprTXID: txID,
	})
	if err != nil {
		return err
	}
	err = s.ModifyYields(db, 0, txID, endTime)
	if err != nil {
		return err
	}
	return nil
}

func (s *samaState) CalcReward(claimerType byte, address common.Address, endTime uint64) (uint64, uint64, uint64, error) {
	if claimerType == 0 {
		foundation := s.GetFoundationAddress()
		if address != common.HexToAddress(foundation) {
			return 0, 0, 0, fmt.Errorf("address or claimer type  err")
		}
		base, err := s.FoundationReword(address, endTime)
		if err != nil {
			return 0, 0, 0, err
		}
		return base, 0, 0, nil
	}
	staker, exist, err := s.GetStakerMeta(claimerType, address)
	if err != nil {
		return 0, 0, 0, err
	}
	if !exist {
		return 0, 0, 0, fmt.Errorf("GetStakerMeta err")
	}
	if endTime < staker.StakeTime { //|| endTime-staker.StakeTime < Seconds7Day {
		return 0, 0, 0, fmt.Errorf("endTime err %d %d", endTime, staker.StakeTime)
	}
	stakeNum := s.StakersNum(claimerType)

	return s.StakerReword(byte(staker.StakerType), stakeNum, staker.StakerAddr, staker.StakeTime, endTime)
}

func (s *samaState) DealStakeTx(db database.Database, staker *StakerMeta) error {
	err := s.UpdateStakerReward(db, byte(staker.StakerType), staker.StakerAddr, staker.TxID, staker.StakeTime)
	if err != nil {
		return err
	}

	err = s.PutStaker(db, staker)
	return err
}

func (s *samaState) DealUnStakeTx(db database.Database, stakerType byte, address common.Address, txID ids.ID, endTime uint64) error {
	err := s.UpdateStakerReward(db, stakerType, address, txID, endTime)
	if err != nil {
		return err
	}

	err = s.DelStaker(db, stakerType, address)
	return err
}

func (s *samaState) CheckPayAmount(db database.Database, userType uint64, amount uint64, startTime uint64, endTime uint64) (bool, error) {
	if (endTime-startTime)%SecondsMonth != 0 {
		return false, nil
	}

	pmeta, exist, _ := s.GetUserType(db, userType)
	if !exist {
		return false, nil
	}
	if amount == pmeta.FeeUnits {
		return true, nil
	}
	return false, nil
}

func (s *samaState) DealAddUserTx(db database.Database, txID ids.ID, blkTime uint64, user *UserMeta) error {
	pmate, exist, err := s.GetUserMeta(db, user.Address)
	if err != nil {
		return err
	}
	if exist {
		if user.StartTime > pmate.EndTime+SecondsMinute || user.StartTime < pmate.EndTime-SecondsMinute {
			return fmt.Errorf("start time err")
		}
		user.StartTime = pmate.StartTime
		user.TxsID = pmate.TxsID
	}
	user.TxsID = append(user.TxsID, txID)
	err = s.PutUser(db, user)
	if err != nil {
		return err
	}
	perc := s.GetPercBurn()
	yield := user.PayAmount * uint64(100-perc) / 100
	err = s.ModifyYields(db, yield, txID, blkTime)
	return err
}

func (s *samaState) UpdateNodeParams(db database.Database, detail *DetailMeta) error {
	s.PutDetail(db, detail.WorkAddress, detail)
	return nil
}

func (s *samaState) IsBeConfirmed(actionType uint64, key string) (bool, ids.ShortID, error) {
	votersNum, actionID, err := s.GetVotersNum(actionType, key)
	if err != nil {
		return false, ids.ShortID{}, err
	}
	routeNum := s.StakersNum(stakerTypeRoute)
	if routeNum < 3 {
		action, _, err := s.GetActionMeta(actionID)
		if err != nil {
			return false, ids.ShortID{}, err
		}
		sroot := s.GetRootAddress()
		for _, voter := range action.Voters {
			if voter == common.HexToAddress(sroot) {
				return true, actionID, nil
			}
		}
	} else {
		if votersNum >= routeNum*2/3 {
			return true, actionID, nil
		}
	}
	return false, ids.ShortID{}, nil
}

func (s *samaState) ProposalStatus(actionID ids.ShortID) (bool, error) {
	action, exist, err := s.GetActionMeta(actionID)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, fmt.Errorf("not found")
	}
	votersNum := len(action.Voters)
	routeNum := s.StakersNum(stakerTypeRoute)
	if routeNum < 3 {
		action, _, err := s.GetActionMeta(actionID)
		if err != nil {
			return false, err
		}
		sroot := s.GetRootAddress()
		for _, voter := range action.Voters {
			if voter == common.HexToAddress(sroot) {
				return true, nil
			}
		}
	} else {
		if votersNum >= routeNum*2/3 {
			return true, nil
		}
	}
	return false, nil
}

func (s *samaState) CheckClaimAddress(address common.Address) (bool, byte, error) {
	ok, stakerType, _ := s.IsStaker(address)
	if ok {
		return true, stakerType, nil
	}
	foundation := s.GetFoundationAddress()
	if address == common.HexToAddress(foundation) {
		return true, 0, nil
	}

	return false, 0, nil
}
