// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"strconv"

	"github.com/SamaNetwork/SamaVM/tdata"
)

var _ UnsignedTransaction = &RefreshTx{}

type RefreshTx struct {
	*BaseTx   `serialize:"true" json:"baseTx"`
	Country   string `serialize:"true" json:"country"`
	LocalIP   string `serialize:"true" json:"localIP"`
	PublicIP  string `serialize:"true" json:"publicIP"`
	MinPort   uint64 `serialize:"true" json:"minPort"`
	MaxPort   uint64 `serialize:"true" json:"maxPort"`
	CheckPort uint64 `serialize:"true" json:"checkPort"`
	WorkKey   string `serialize:"true" json:"workKey"`
}

func IsRepeated(d *DetailMeta, r *RefreshTx) bool {
	if d.LocalIP == r.LocalIP && d.MinPort == r.MinPort && d.MaxPort == r.MaxPort && d.PublicIP == r.PublicIP &&
		d.CheckPort == r.CheckPort && d.Country == r.Country && d.WorkKey == r.WorkKey {
		return true
	}
	return false
}

func (r *RefreshTx) Execute(t *TransactionContext) error {
	samaState := t.vm.SamaState()
	ok, _, err := samaState.IsValidWorkAddress(t.Sender)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("sender must route or ser")
	}

	pmate, exist, err := samaState.GetDetailMeta(t.Sender)
	if err != nil {
		return err
	}
	if exist {
		if IsRepeated(pmate, r) {
			return fmt.Errorf("is repeated")
		}
	} else {
		return fmt.Errorf("not found")
	}

	err = samaState.PutDetail(t.Database, t.Sender, &DetailMeta{
		StakerType:     pmate.StakerType,
		Country:        r.Country,
		WorkKey:        r.WorkKey,
		LocalIP:        r.LocalIP,
		MinPort:        r.MinPort,
		MaxPort:        r.MaxPort,
		PublicIP:       r.PublicIP,
		CheckPort:      r.CheckPort,
		TxID:           t.TxID,
		LastUpdateTime: t.BlockTime,
		WorkAddress:    t.Sender,
		StakeAddress:   pmate.StakeAddress,
	})

	return err
}

func (r *RefreshTx) FeeUnits(g *Genesis) uint64 {
	return 0 //r.BaseTx.FeeUnits(g) //+ valueUnits(g, uint64(len(r.Value)))
}

func (r *RefreshTx) LoadUnits(g *Genesis) uint64 {
	return 0 //r.FeeUnits(g)
}

func (r *RefreshTx) Copy() UnsignedTransaction {
	return &RefreshTx{
		BaseTx:    r.BaseTx.Copy(),
		Country:   r.Country,
		WorkKey:   r.WorkKey,
		LocalIP:   r.LocalIP,
		MinPort:   r.MinPort,
		MaxPort:   r.MaxPort,
		PublicIP:  r.PublicIP,
		CheckPort: r.CheckPort,
	}
}

func (r *RefreshTx) TypedData() *tdata.TypedData {
	return tdata.CreateTypedData(
		r.Magic, Refresh,
		[]tdata.Type{
			{Name: tdCountry, Type: tdString},
			{Name: tdWorkKey, Type: tdString},
			{Name: tdLocalIP, Type: tdString},
			{Name: tdMinPort, Type: tdUint64},
			{Name: tdMaxPort, Type: tdUint64},
			{Name: tdPublicIP, Type: tdString},
			{Name: tdCheckPort, Type: tdUint64},
			{Name: tdPrice, Type: tdUint64},
			{Name: tdBlockID, Type: tdString},
		},
		tdata.TypedDataMessage{
			tdCountry:   r.Country,
			tdWorkKey:   r.WorkKey,
			tdLocalIP:   r.LocalIP,
			tdMinPort:   strconv.FormatUint(r.MinPort, 10),
			tdMaxPort:   strconv.FormatUint(r.MaxPort, 10),
			tdPublicIP:  r.PublicIP,
			tdCheckPort: strconv.FormatUint(r.CheckPort, 10),
			tdPrice:     strconv.FormatUint(r.Price, 10),
			tdBlockID:   r.BlockID.String(),
		},
	)
}

func (r *RefreshTx) Activity() *Activity {
	return &Activity{
		Typ:       Refresh,
		Country:   r.Country,
		WorkKey:   r.WorkKey,
		LocalIP:   r.LocalIP,
		MinPort:   r.MinPort,
		MaxPort:   r.MaxPort,
		PublicIP:  r.PublicIP,
		CheckPort: r.CheckPort,
	}
}
