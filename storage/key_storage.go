// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/coinbase/rosetta-sdk-go/keys"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// WARNING: KEY STORAGE USING THIS PACKAGE IS NOT SECURE!!!! ONLY USE
// FOR TESTING!!!!

// PrefundedAccount is used to load prefunded addresses into key storage.
type PrefundedAccount struct {
	PrivateKeyHex string          `json:"privkey"`
	Address       string          `json:"address"`
	CurveType     types.CurveType `json:"curve_type"`
	Currency      *types.Currency `json:"currency"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

const (
	keyNamespace = "key"
)

func getAddressKey(address string) []byte {
	return []byte(
		fmt.Sprintf("%s/%s", keyNamespace, address),
	)
}

// KeyStorage implements key storage methods
// on top of a Database and DatabaseTransaction interface.
type KeyStorage struct {
	db Database
}

// NewKeyStorage returns a new KeyStorage.
func NewKeyStorage(
	db Database,
) *KeyStorage {
	return &KeyStorage{
		db: db,
	}
}

// Key is the struct stored in key storage. This
// is public so that accounts can be loaded from
// a configuration file.
type Key struct {
	Address string        `json:"address"`
	KeyPair *keys.KeyPair `json:"keypair"`
}

// StoreTransactional stores a key in a database transaction.
func (k *KeyStorage) StoreTransactional(
	ctx context.Context,
	address string,
	keyPair *keys.KeyPair,
	dbTx DatabaseTransaction,
) error {
	exists, _, err := dbTx.Get(ctx, getAddressKey(address))
	if err != nil {
		return fmt.Errorf("%w: %s. %v", ErrAddrCheckIfExistsFailed, address, err)
	}

	if exists {
		return fmt.Errorf("%w: address %s already exists", ErrAddrExists, address)
	}

	val, err := k.db.Compressor().Encode("", &Key{
		Address: address,
		KeyPair: keyPair,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeKeyFailed, err)
	}

	err = dbTx.Set(ctx, getAddressKey(address), val, true)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStoreKeyFailed, err)
	}

	return nil
}

// Store saves a keys.KeyPair for a given address. If the address already
// exists, an error is returned.
func (k *KeyStorage) Store(
	ctx context.Context,
	address string,
	keyPair *keys.KeyPair,
) error {
	dbTx := k.db.NewDatabaseTransaction(ctx, true)
	defer dbTx.Discard(ctx)

	if err := k.StoreTransactional(ctx, address, keyPair, dbTx); err != nil {
		return fmt.Errorf("%w: unable to store key", err)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrCommitKeyFailed, err)
	}

	return nil
}

// GetTransactional returns a *keys.KeyPair for an address in a DatabaseTransaction, if it exists.
func (k *KeyStorage) GetTransactional(
	ctx context.Context,
	dbTx DatabaseTransaction,
	address string,
) (*keys.KeyPair, error) {
	exists, rawKey, err := dbTx.Get(ctx, getAddressKey(address))
	if err != nil {
		return nil, fmt.Errorf("%w: %s. %v", ErrAddrGetFailed, address, err)
	}

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrAddrNotFound, address)
	}

	var kp Key
	if err := k.db.Compressor().Decode("", rawKey, &kp, true); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseKeyPairFailed, err)
	}

	return kp.KeyPair, nil
}

// Get returns a *keys.KeyPair for an address, if it exists.
func (k *KeyStorage) Get(ctx context.Context, address string) (*keys.KeyPair, error) {
	transaction := k.db.NewDatabaseTransaction(ctx, false)
	defer transaction.Discard(ctx)

	return k.GetTransactional(ctx, transaction, address)
}

// GetAllAddressesTransactional returns all addresses in key storage.
func (k *KeyStorage) GetAllAddressesTransactional(
	ctx context.Context,
	dbTx DatabaseTransaction,
) ([]string, error) {
	addresses := []string{}
	_, err := dbTx.Scan(
		ctx,
		[]byte(keyNamespace),
		func(key []byte, v []byte) error {
			var kp Key
			if err := k.db.Compressor().Decode("", v, &kp, false); err != nil {
				return fmt.Errorf("%w: %v", ErrKeyScanFailed, err)
			}

			addresses = append(addresses, kp.Address)
			return nil
		},
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyScanFailed, err)
	}

	return addresses, nil
}

// GetAllAddresses returns all addresses in key storage.
func (k *KeyStorage) GetAllAddresses(ctx context.Context) ([]string, error) {
	dbTx := k.db.NewDatabaseTransaction(ctx, false)
	defer dbTx.Discard(ctx)

	return k.GetAllAddressesTransactional(ctx, dbTx)
}

// Sign attempts to sign a slice of *types.SigningPayload with the keys in KeyStorage.
func (k *KeyStorage) Sign(
	ctx context.Context,
	payloads []*types.SigningPayload,
) ([]*types.Signature, error) {
	signatures := make([]*types.Signature, len(payloads))
	for i, payload := range payloads {
		keyPair, err := k.Get(ctx, payload.Address)
		if err != nil {
			return nil, fmt.Errorf("%w for %s: %v", ErrKeyGetFailed, payload.Address, err)
		}

		signer, err := keyPair.Signer()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSignerCreateFailed, err)
		}

		if len(payload.SignatureType) == 0 {
			return nil, fmt.Errorf("%w %d", ErrDetermineSigTypeFailed, i)
		}

		signature, err := signer.Sign(payload, payload.SignatureType)
		if err != nil {
			return nil, fmt.Errorf("%w for %d: %v", ErrSignPayloadFailed, i, err)
		}

		signatures[i] = signature
	}

	return signatures, nil
}

// RandomAddress returns a random address from all addresses.
func (k *KeyStorage) RandomAddress(ctx context.Context) (string, error) {
	addresses, err := k.GetAllAddresses(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAddrsGetAllFailed, err)
	}

	if len(addresses) == 0 {
		return "", ErrNoAddrAvailable
	}

	return addresses[rand.Intn(len(addresses))], nil
}

// ImportAccounts loads a set of prefunded accounts into key storage.
func (k *KeyStorage) ImportAccounts(ctx context.Context, accounts []*PrefundedAccount) error {
	// Import prefunded account and save to database
	for _, acc := range accounts {
		keyPair, err := keys.ImportPrivKey(acc.PrivateKeyHex, acc.CurveType)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrAddrImportFailed, err)
		}

		// Skip if key already exists
		err = k.Store(ctx, acc.Address, keyPair)
		if errors.Is(err, ErrAddrExists) {
			continue
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPrefundedAcctStoreFailed, err)
		}
	}
	return nil
}
