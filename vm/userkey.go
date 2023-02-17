// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"fmt"

	"github.com/SamaNetwork/SamaVM/chain"
	"github.com/ava-labs/avalanchego/database/encdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// Key in the database whose corresponding value is the list of
// addresses this userKey controls
var addressesKey = ids.Empty[:]

var (
	errDBNil  = errors.New("db uninitialized")
	errKeyNil = errors.New("key uninitialized")
)

type userKey struct {
	secpFactory *crypto.FactorySECP256K1R
	// This userKey's database, acquired from the keystore
	db *encdb.Database
}

// Get the addresses controlled by this userKey
func (u *userKey) getAddresses() ([]common.Address, error) {
	if u.db == nil {
		return nil, errDBNil
	}

	// If userKey has no addresses, return empty list
	hasAddresses, err := u.db.Has(addressesKey)
	if err != nil {
		return nil, err
	}
	if !hasAddresses {
		return nil, nil
	}

	// User has addresses. Get them.
	bytes, err := u.db.Get(addressesKey)
	if err != nil {
		return nil, err
	}
	addresses := []common.Address{}

	if _, err := chain.Unmarshal(bytes, &addresses); err != nil {
		return nil, err
	}
	return addresses, nil
}

// controlsAddress returns true iff this userKey controls the given address
func (u *userKey) controlsAddress(address common.Address) (bool, error) {
	if u.db == nil {
		return false, errDBNil
		//} else if address.IsZero() {
		//	return false, errEmptyAddress
	}
	return u.db.Has(address.Bytes())
}

// putAddress persists that this userKey controls address controlled by [privKey]
func (u *userKey) putAddress(privKey *crypto.PrivateKeySECP256K1R) error {
	if privKey == nil {
		return errKeyNil
	}

	pubKey := privKey.PublicKey().(*crypto.PublicKeySECP256K1R)
	address := ethcrypto.PubkeyToAddress(*(pubKey.ToECDSA()))

	controlsAddress, err := u.controlsAddress(address)
	if err != nil {
		return err
	}
	if controlsAddress { // userKey already controls this address. Do nothing.
		return nil
	}

	if err := u.db.Put(address.Bytes(), privKey.Bytes()); err != nil { // Address --> private key
		return err
	}

	addresses := make([]common.Address, 0) // Add address to list of addresses userKey controls
	userHasAddresses, err := u.db.Has(addressesKey)
	if err != nil {
		return err
	}
	if userHasAddresses { // Get addresses this userKey already controls, if they exist
		if addresses, err = u.getAddresses(); err != nil {
			return err
		}
	}
	addresses = append(addresses, address)

	bytes, err := chain.Marshal(addresses)
	if err != nil {
		return err
	}
	if err := u.db.Put(addressesKey, bytes); err != nil {
		return err
	}
	return nil
}

// Key returns the private key that controls the given address
func (u *userKey) getKey(address common.Address) (*crypto.PrivateKeySECP256K1R, error) {
	if u.db == nil {
		return nil, errDBNil
		//} else if address.IsZero() {
		//	return nil, errEmptyAddress
	}

	bytes, err := u.db.Get(address.Bytes())
	if err != nil {
		return nil, err
	}
	sk, err := u.secpFactory.ToPrivateKey(bytes)
	if err != nil {
		return nil, err
	}
	if sk, ok := sk.(*crypto.PrivateKeySECP256K1R); ok {
		return sk, nil
	}
	return nil, fmt.Errorf("expected private key to be type *crypto.PrivateKeySECP256K1R but is type %T", sk)
}

// Return all private keys controlled by this userKey
func (u *userKey) getKeys() ([]*crypto.PrivateKeySECP256K1R, error) {
	addrs, err := u.getAddresses()
	if err != nil {
		return nil, err
	}
	keys := make([]*crypto.PrivateKeySECP256K1R, len(addrs))
	for i, addr := range addrs {
		key, err := u.getKey(addr)
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}
	return keys, nil
}
