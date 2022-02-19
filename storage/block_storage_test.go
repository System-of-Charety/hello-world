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
	"fmt"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	"github.com/stretchr/testify/assert"
)

func TestHeadBlockIdentifier(t *testing.T) {
	var (
		newBlockIdentifier = &types.BlockIdentifier{
			Hash:  "blah",
			Index: 0,
		}
		newBlockIdentifier2 = &types.BlockIdentifier{
			Hash:  "blah2",
			Index: 1,
		}
	)

	ctx := context.Background()

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)

	database, err := NewBadgerStorage(ctx, newDir)
	assert.NoError(t, err)
	defer func() {
		database.Close(ctx)
		utils.RemoveTempDir(newDir)
	}()

	storage := NewBlockStorage(database)

	t.Run("No head block set", func(t *testing.T) {
		blockIdentifier, err := storage.GetHeadBlockIdentifier(ctx)
		assert.EqualError(t, err, ErrHeadBlockNotFound.Error())
		assert.Nil(t, blockIdentifier)
	})

	t.Run("Set and get head block", func(t *testing.T) {
		txn := storage.db.NewDatabaseTransaction(ctx, true)
		assert.NoError(t, storage.StoreHeadBlockIdentifier(ctx, txn, newBlockIdentifier))
		assert.NoError(t, txn.Commit(ctx))

		blockIdentifier, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, newBlockIdentifier, blockIdentifier)
	})

	t.Run("Discard head block update", func(t *testing.T) {
		txn := storage.db.NewDatabaseTransaction(ctx, true)
		assert.NoError(t, storage.StoreHeadBlockIdentifier(ctx, txn,
			&types.BlockIdentifier{
				Hash:  "no blah",
				Index: 10,
			}),
		)
		txn.Discard(ctx)

		blockIdentifier, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, newBlockIdentifier, blockIdentifier)
	})

	t.Run("Multiple updates to head block", func(t *testing.T) {
		txn := storage.db.NewDatabaseTransaction(ctx, true)
		assert.NoError(t, storage.StoreHeadBlockIdentifier(ctx, txn, newBlockIdentifier2))
		assert.NoError(t, txn.Commit(ctx))

		blockIdentifier, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		txn.Discard(ctx)
		assert.Equal(t, newBlockIdentifier2, blockIdentifier)
	})
}

func simpleTransactionFactory(
	hash string,
	address string,
	value string,
	currency *types.Currency,
) *types.Transaction {
	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hash,
		},
		Operations: []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 0,
				},
				Type:   "Transfer",
				Status: "Success",
				Account: &types.AccountIdentifier{
					Address: address,
				},
				Amount: &types.Amount{
					Value:    value,
					Currency: currency,
				},
			},
		},
	}
}

var (
	genesisBlock = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 0",
			Index: 0,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 0",
			Index: 0,
		},
		Timestamp: 1,
	}

	newBlock = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 1",
			Index: 1,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 0",
			Index: 0,
		},
		Timestamp: 1,
		Transactions: []*types.Transaction{
			simpleTransactionFactory("blahTx", "addr1", "100", &types.Currency{Symbol: "hello"}),
		},
	}

	badBlockIdentifier = &types.BlockIdentifier{
		Hash:  "missing blah",
		Index: 0,
	}

	newBlock2 = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 2",
			Index: 2,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 1",
			Index: 1,
		},
		Timestamp: 1,
		Transactions: []*types.Transaction{
			simpleTransactionFactory("blahTx", "addr1", "100", &types.Currency{Symbol: "hello"}),
		},
	}
	lazyBlock2 = &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "blah 2",
				Index: 2,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "blah 1",
				Index: 1,
			},
			Timestamp: 1,
		},
		OtherTransactions: []*types.TransactionIdentifier{
			{
				Hash: "blahTx",
			},
		},
	}

	complexBlock = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 3",
			Index: 3,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 2",
			Index: 2,
		},
		Timestamp: 1,
		Transactions: []*types.Transaction{
			{
				TransactionIdentifier: &types.TransactionIdentifier{
					Hash: "blahTx 2",
				},
				Operations: []*types.Operation{
					{
						OperationIdentifier: &types.OperationIdentifier{
							Index: 0,
						},
						Type:   "Transfer",
						Status: "Success",
						Account: &types.AccountIdentifier{
							Address: "addr1",
							SubAccount: &types.SubAccountIdentifier{
								Address: "staking",
								Metadata: map[string]interface{}{
									"other_complex_stuff": []interface{}{
										map[string]interface{}{
											"neat": "test",
											"more complex": map[string]interface{}{
												"neater": "testier",
											},
										},
										map[string]interface{}{
											"i love": "ice cream",
										},
									},
								},
							},
						},
						Amount: &types.Amount{
							Value: "100",
							Currency: &types.Currency{
								Symbol: "hello",
							},
						},
					},
				},
				Metadata: map[string]interface{}{
					"other_stuff":  []interface{}{"stuff"},
					"simple_stuff": "abc",
					"super_complex_stuff": map[string]interface{}{
						"neat": "test",
						"more complex": map[string]interface{}{
							"neater": "testier",
						},
					},
				},
			},
		},
	}

	duplicateTxBlock = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 4",
			Index: 4,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 3",
			Index: 3,
		},
		Timestamp: 1,
		Transactions: []*types.Transaction{
			simpleTransactionFactory("blahTx3", "addr2", "200", &types.Currency{Symbol: "hello"}),
			simpleTransactionFactory("blahTx3", "addr2", "200", &types.Currency{Symbol: "hello"}),
		},
	}

	gapBlock = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 10",
			Index: 10,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "blah 3",
			Index: 3,
		},
	}
)

func findTransactionWithDbTransaction(
	ctx context.Context,
	storage *BlockStorage,
	transactionIdentifier *types.TransactionIdentifier,
) (*types.BlockIdentifier, *types.Transaction, error) {
	txn := storage.db.NewDatabaseTransaction(ctx, false)
	defer txn.Discard(ctx)

	return storage.FindTransaction(
		ctx,
		transactionIdentifier,
		txn,
	)
}

func TestBlock(t *testing.T) {
	ctx := context.Background()

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)

	database, err := NewBadgerStorage(ctx, newDir)
	assert.NoError(t, err)
	defer func() {
		database.Close(ctx)
		utils.RemoveTempDir(newDir)
	}()

	storage := NewBlockStorage(database)

	t.Run("Get non-existent tx", func(t *testing.T) {
		newestBlock, transaction, err := findTransactionWithDbTransaction(
			ctx,
			storage,
			newBlock.Transactions[0].TransactionIdentifier,
		)
		assert.NoError(t, err)
		assert.Nil(t, newestBlock)
		assert.Nil(t, transaction)
	})

	t.Run("Set and get block", func(t *testing.T) {
		err := storage.AddBlock(ctx, newBlock)
		assert.NoError(t, err)

		block, err := storage.GetBlock(
			ctx,
			types.ConstructPartialBlockIdentifier(newBlock.BlockIdentifier),
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock, block)

		block, err = storage.GetBlock(
			ctx,
			&types.PartialBlockIdentifier{Index: &newBlock.BlockIdentifier.Index},
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock, block)

		block, err = storage.GetBlock(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, newBlock, block)

		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, newBlock.BlockIdentifier, head)

		newestBlock, transaction, err := findTransactionWithDbTransaction(
			ctx,
			storage,
			newBlock.Transactions[0].TransactionIdentifier,
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock.BlockIdentifier, newestBlock)
		assert.Equal(t, newBlock.Transactions[0], transaction)
	})

	t.Run("Get non-existent block", func(t *testing.T) {
		identifier := types.ConstructPartialBlockIdentifier(badBlockIdentifier)
		block, err := storage.GetBlock(ctx, identifier)
		assert.EqualError(
			t,
			err,
			fmt.Errorf("%w: %+v", ErrBlockNotFound, identifier).Error(),
		)
		assert.Nil(t, block)
	})

	t.Run("Get non-existent block index", func(t *testing.T) {
		badIndex := int64(100000)
		identifier := &types.PartialBlockIdentifier{Index: &badIndex}
		block, err := storage.GetBlock(ctx, identifier)
		assert.EqualError(
			t,
			err,
			fmt.Errorf("%w: %+v", ErrBlockNotFound, identifier).Error(),
		)
		assert.Nil(t, block)
	})

	t.Run("Set duplicate block hash", func(t *testing.T) {
		err = storage.AddBlock(ctx, newBlock)
		assert.Contains(t, err.Error(), ErrDuplicateKey.Error())
	})

	t.Run("Set duplicate transaction hash (from prior block)", func(t *testing.T) {
		err = storage.AddBlock(ctx, newBlock2)
		assert.NoError(t, err)

		block, err := storage.GetBlock(
			ctx,
			types.ConstructPartialBlockIdentifier(newBlock2.BlockIdentifier),
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2, block)

		block, err = storage.GetBlock(
			ctx,
			&types.PartialBlockIdentifier{Index: &newBlock2.BlockIdentifier.Index},
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2, block)

		block, err = storage.GetBlock(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2, block)

		blockLazy, err := storage.GetBlockLazy(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, lazyBlock2, blockLazy)

		blockTransaction, err := storage.GetBlockTransaction(
			ctx,
			blockLazy.Block.BlockIdentifier,
			blockLazy.OtherTransactions[0],
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.Transactions[0], blockTransaction)

		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.BlockIdentifier, head)

		newestBlock, transaction, err := findTransactionWithDbTransaction(
			ctx,
			storage,
			newBlock.Transactions[0].TransactionIdentifier,
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.BlockIdentifier, newestBlock)
		assert.Equal(t, newBlock2.Transactions[0], transaction)
	})

	t.Run("Remove block and re-set block of same hash", func(t *testing.T) {
		err := storage.RemoveBlock(ctx, newBlock2.BlockIdentifier)
		assert.NoError(t, err)

		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.ParentBlockIdentifier, head)

		err = storage.AddBlock(ctx, newBlock2)
		assert.NoError(t, err)

		head, err = storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.BlockIdentifier, head)

		newestBlock, transaction, err := findTransactionWithDbTransaction(
			ctx,
			storage,
			newBlock.Transactions[0].TransactionIdentifier,
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.BlockIdentifier, newestBlock)
		assert.Equal(t, newBlock2.Transactions[0], transaction)
	})

	t.Run("Add block with complex metadata", func(t *testing.T) {
		err := storage.AddBlock(ctx, complexBlock)
		assert.NoError(t, err)

		block, err := storage.GetBlock(
			ctx,
			types.ConstructPartialBlockIdentifier(complexBlock.BlockIdentifier),
		)
		assert.NoError(t, err)
		assert.Equal(t, complexBlock, block)

		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, complexBlock.BlockIdentifier, head)

		newestBlock, transaction, err := findTransactionWithDbTransaction(
			ctx,
			storage,
			newBlock.Transactions[0].TransactionIdentifier,
		)
		assert.NoError(t, err)
		assert.Equal(t, newBlock2.BlockIdentifier, newestBlock)
		assert.Equal(t, newBlock2.Transactions[0], transaction)
	})

	t.Run("Set duplicate transaction hash (same block)", func(t *testing.T) {
		err = storage.AddBlock(ctx, duplicateTxBlock)
		assert.Contains(t, err.Error(), ErrDuplicateTransactionHash.Error())

		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, complexBlock.BlockIdentifier, head)
	})

	t.Run("Add block after omitted", func(t *testing.T) {
		err := storage.AddBlock(ctx, gapBlock)
		assert.NoError(t, err)

		block, err := storage.GetBlock(
			ctx,
			types.ConstructPartialBlockIdentifier(gapBlock.BlockIdentifier),
		)
		assert.NoError(t, err)
		assert.Equal(t, gapBlock, block)

		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, gapBlock.BlockIdentifier, head)
	})
}

func TestManyBlocks(t *testing.T) {
	ctx := context.Background()

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)

	database, err := NewBadgerStorage(ctx, newDir)
	assert.NoError(t, err)
	defer func() {
		database.Close(ctx)
		utils.RemoveTempDir(newDir)
	}()

	storage := NewBlockStorage(database)

	for i := int64(0); i < 10000; i++ {
		blockIdentifier := &types.BlockIdentifier{
			Index: i,
			Hash:  fmt.Sprintf("block %d", i),
		}
		parentBlockIndex := blockIdentifier.Index - 1
		if parentBlockIndex < 0 {
			parentBlockIndex = 0
		}
		parentBlockIdentifier := &types.BlockIdentifier{
			Index: parentBlockIndex,
			Hash:  fmt.Sprintf("block %d", parentBlockIndex),
		}

		block := &types.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
		}

		assert.NoError(t, storage.AddBlock(ctx, block))
		head, err := storage.GetHeadBlockIdentifier(ctx)
		assert.NoError(t, err)
		assert.Equal(t, blockIdentifier, head)
	}
}

func TestCreateBlockCache(t *testing.T) {
	ctx := context.Background()

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)

	database, err := NewBadgerStorage(ctx, newDir)
	assert.NoError(t, err)
	defer func() {
		database.Close(ctx)
		utils.RemoveTempDir(newDir)
	}()

	storage := NewBlockStorage(database)

	t.Run("no blocks processed", func(t *testing.T) {
		assert.Equal(t, []*types.BlockIdentifier{}, storage.CreateBlockCache(ctx))
	})

	t.Run("1 block processed", func(t *testing.T) {
		err = storage.AddBlock(ctx, genesisBlock)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]*types.BlockIdentifier{genesisBlock.BlockIdentifier},
			storage.CreateBlockCache(ctx),
		)
	})

	t.Run("2 blocks processed", func(t *testing.T) {
		err = storage.AddBlock(ctx, newBlock)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]*types.BlockIdentifier{genesisBlock.BlockIdentifier, newBlock.BlockIdentifier},
			storage.CreateBlockCache(ctx),
		)
	})

	t.Run("3 blocks processed (with gap)", func(t *testing.T) {
		simpleGap := &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "block 100",
				Index: 100,
			},
			ParentBlockIdentifier: newBlock.BlockIdentifier,
		}

		err = storage.AddBlock(ctx, simpleGap)
		assert.NoError(t, err)
		assert.Equal(
			t,
			[]*types.BlockIdentifier{
				genesisBlock.BlockIdentifier,
				newBlock.BlockIdentifier,
				simpleGap.BlockIdentifier,
			},
			storage.CreateBlockCache(ctx),
		)
	})
}

func TestAtTip(t *testing.T) {
	ctx := context.Background()

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)

	database, err := NewBadgerStorage(ctx, newDir)
	assert.NoError(t, err)
	defer func() {
		database.Close(ctx)
		utils.RemoveTempDir(newDir)
	}()

	storage := NewBlockStorage(database)
	tipDelay := int64(100)

	t.Run("no blocks processed", func(t *testing.T) {
		atTip, blockIdentifier, err := storage.AtTip(ctx, tipDelay)
		assert.NoError(t, err)
		assert.False(t, atTip)
		assert.Nil(t, blockIdentifier)
	})

	t.Run("Add old block", func(t *testing.T) {
		err := storage.AddBlock(ctx, &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "block 0",
				Index: 0,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "block 0",
				Index: 0,
			},
			Timestamp: utils.Milliseconds() - (3 * tipDelay * utils.MillisecondsInSecond),
		})
		assert.NoError(t, err)

		atTip, blockIdentifier, err := storage.AtTip(ctx, tipDelay)
		assert.NoError(t, err)
		assert.False(t, atTip)
		assert.Nil(t, blockIdentifier)
	})

	t.Run("Add new block", func(t *testing.T) {
		err := storage.AddBlock(ctx, &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "block 1",
				Index: 1,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "block 0",
				Index: 0,
			},
			Timestamp: utils.Milliseconds(),
		})
		assert.NoError(t, err)

		atTip, blockIdentifier, err := storage.AtTip(ctx, tipDelay)
		assert.NoError(t, err)
		assert.True(t, atTip)
		assert.Equal(t, &types.BlockIdentifier{
			Hash:  "block 1",
			Index: 1,
		}, blockIdentifier)
	})
}
