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

package syncer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	mocks "github.com/coinbase/rosetta-sdk-go/mocks/syncer"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	networkIdentifier = &types.NetworkIdentifier{
		Blockchain: "blah",
		Network:    "testnet",
	}

	currency = &types.Currency{
		Symbol:   "Blah",
		Decimals: 2,
	}

	recipient = &types.AccountIdentifier{
		Address: "acct1",
	}

	recipientAmount = &types.Amount{
		Value:    "100",
		Currency: currency,
	}

	recipientOperation = &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: 0,
		},
		Type:    "Transfer",
		Status:  "Success",
		Account: recipient,
		Amount:  recipientAmount,
	}

	recipientFailureOperation = &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: 1,
		},
		Type:    "Transfer",
		Status:  "Failure",
		Account: recipient,
		Amount:  recipientAmount,
	}

	recipientTransaction = &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: "tx1",
		},
		Operations: []*types.Operation{
			recipientOperation,
			recipientFailureOperation,
		},
	}

	sender = &types.AccountIdentifier{
		Address: "acct2",
	}

	senderAmount = &types.Amount{
		Value:    "-100",
		Currency: currency,
	}

	senderOperation = &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: 0,
		},
		Type:    "Transfer",
		Status:  "Success",
		Account: sender,
		Amount:  senderAmount,
	}

	senderTransaction = &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: "tx2",
		},
		Operations: []*types.Operation{
			senderOperation,
		},
	}

	orphanGenesis = &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "1",
			Index: 1,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "0a",
			Index: 0,
		},
		Transactions: []*types.Transaction{},
	}

	blockSequence = []*types.Block{
		{ // genesis
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "0",
				Index: 0,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "0",
				Index: 0,
			},
		},
		{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "1",
				Index: 1,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "0",
				Index: 0,
			},
			Transactions: []*types.Transaction{
				recipientTransaction,
			},
		},
		{ // reorg
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "2",
				Index: 2,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "1a",
				Index: 1,
			},
		},
		{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "1a",
				Index: 1,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "0",
				Index: 0,
			},
		},
		{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "3",
				Index: 3,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "2",
				Index: 2,
			},
			Transactions: []*types.Transaction{
				senderTransaction,
			},
		},
		{ // invalid block
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  "5",
				Index: 5,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  "4",
				Index: 4,
			},
		},
	}
)

func lastBlockIdentifier(syncer *Syncer) *types.BlockIdentifier {
	return syncer.pastBlocks[len(syncer.pastBlocks)-1]
}

func TestProcessBlock(t *testing.T) {
	ctx := context.Background()

	mockHelper := &mocks.Helper{}
	mockHandler := &mocks.Handler{}
	syncer := New(networkIdentifier, mockHelper, mockHandler, nil, WithConcurrency(1))
	syncer.genesisBlock = blockSequence[0].BlockIdentifier

	t.Run("No block exists", func(t *testing.T) {
		mockHandler.On("BlockAdded", ctx, blockSequence[0]).Return(nil).Once()
		assert.Equal(
			t,
			[]*types.BlockIdentifier{},
			syncer.pastBlocks,
		)
		err := syncer.processBlock(
			ctx,
			blockSequence[0],
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), syncer.nextIndex)
		assert.Equal(t, blockSequence[0].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{blockSequence[0].BlockIdentifier},
			syncer.pastBlocks,
		)
	})

	t.Run("Orphan genesis", func(t *testing.T) {
		err := syncer.processBlock(
			ctx,
			orphanGenesis,
		)

		assert.EqualError(t, err, "cannot remove genesis block")
		assert.Equal(t, int64(1), syncer.nextIndex)
		assert.Equal(t, blockSequence[0].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{blockSequence[0].BlockIdentifier},
			syncer.pastBlocks,
		)
	})

	t.Run("Block exists, no reorg", func(t *testing.T) {
		mockHandler.On("BlockAdded", ctx, blockSequence[1]).Return(nil).Once()
		err := syncer.processBlock(
			ctx,
			blockSequence[1],
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), syncer.nextIndex)
		assert.Equal(t, blockSequence[1].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{
				blockSequence[0].BlockIdentifier,
				blockSequence[1].BlockIdentifier,
			},
			syncer.pastBlocks,
		)
	})

	t.Run("Orphan block", func(t *testing.T) {
		mockHandler.On("BlockRemoved", ctx, blockSequence[1].BlockIdentifier).Return(nil).Once()
		err := syncer.processBlock(
			ctx,
			blockSequence[2],
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), syncer.nextIndex)
		assert.Equal(t, blockSequence[0].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{blockSequence[0].BlockIdentifier},
			syncer.pastBlocks,
		)

		mockHandler.On("BlockAdded", ctx, blockSequence[3]).Return(nil).Once()
		err = syncer.processBlock(
			ctx,
			blockSequence[3],
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), syncer.nextIndex)
		assert.Equal(t, blockSequence[3].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{
				blockSequence[0].BlockIdentifier,
				blockSequence[3].BlockIdentifier,
			},
			syncer.pastBlocks,
		)

		mockHandler.On("BlockAdded", ctx, blockSequence[2]).Return(nil).Once()
		err = syncer.processBlock(
			ctx,
			blockSequence[2],
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), syncer.nextIndex)
		assert.Equal(t, blockSequence[2].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{
				blockSequence[0].BlockIdentifier,
				blockSequence[3].BlockIdentifier,
				blockSequence[2].BlockIdentifier,
			},
			syncer.pastBlocks,
		)
	})

	t.Run("Out of order block", func(t *testing.T) {
		err := syncer.processBlock(
			ctx,
			blockSequence[5],
		)
		assert.Contains(t, err.Error(), "got block 5 instead of 3")
		assert.Equal(t, int64(3), syncer.nextIndex)
		assert.Equal(t, blockSequence[2].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{
				blockSequence[0].BlockIdentifier,
				blockSequence[3].BlockIdentifier,
				blockSequence[2].BlockIdentifier,
			},
			syncer.pastBlocks,
		)
	})

	t.Run("Process omitted block", func(t *testing.T) {
		err := syncer.processBlock(
			ctx,
			nil,
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), syncer.nextIndex)
		assert.Equal(t, blockSequence[2].BlockIdentifier, lastBlockIdentifier(syncer))
		assert.Equal(
			t,
			[]*types.BlockIdentifier{
				blockSequence[0].BlockIdentifier,
				blockSequence[3].BlockIdentifier,
				blockSequence[2].BlockIdentifier,
			},
			syncer.pastBlocks,
		)
	})

	mockHelper.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func createBlocks(startIndex int64, endIndex int64, add string) []*types.Block {
	blocks := []*types.Block{}
	for i := startIndex; i <= endIndex; i++ {
		parentIndex := i - 1
		if parentIndex < 0 {
			parentIndex = 0
		}

		blocks = append(blocks, &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Hash:  fmt.Sprintf("block %s%d", add, i),
				Index: i,
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Hash:  fmt.Sprintf("block %s%d", add, parentIndex),
				Index: parentIndex,
			},
		})
	}

	return blocks
}

func assertNotCanceled(t *testing.T, args mock.Arguments) {
	err := args.Get(0).(context.Context)
	assert.NoError(t, err.Err())
}

func TestSync_NoReorg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockHelper := &mocks.Helper{}
	mockHandler := &mocks.Handler{}
	syncer := New(networkIdentifier, mockHelper, mockHandler, cancel, WithConcurrency(16))

	// Force syncer to only get part of the way through the full range
	mockHelper.On("NetworkStatus", ctx, networkIdentifier).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 200",
			Index: 200,
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 0",
			Index: 0,
		},
	}, nil).Twice()

	mockHelper.On("NetworkStatus", ctx, networkIdentifier).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 1300",
			Index: 1300,
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 0",
			Index: 0,
		},
	}, nil).Twice()

	blocks := createBlocks(0, 1200, "")

	// Create a block gap
	blocks[100] = nil
	blocks[101].ParentBlockIdentifier = blocks[99].BlockIdentifier
	for i, b := range blocks {
		index := int64(i)
		mockHelper.On(
			"Block",
			mock.AnythingOfType("*context.cancelCtx"),
			networkIdentifier,
			&types.PartialBlockIdentifier{Index: &index},
		).Return(
			b,
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()

		if b == nil {
			continue
		}

		mockHandler.On(
			"BlockAdded",
			mock.AnythingOfType("*context.cancelCtx"),
			b,
		).Return(
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
	}

	err := syncer.Sync(ctx, -1, 1200)
	assert.NoError(t, err)
	mockHelper.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func TestSync_SpecificStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockHelper := &mocks.Helper{}
	mockHandler := &mocks.Handler{}
	syncer := New(networkIdentifier, mockHelper, mockHandler, cancel, WithConcurrency(16))

	mockHelper.On("NetworkStatus", ctx, networkIdentifier).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 1300",
			Index: 1300,
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 0",
			Index: 0,
		},
	}, nil)

	blocks := createBlocks(100, 1200, "")
	for _, b := range blocks {
		mockHelper.On(
			"Block",
			mock.AnythingOfType("*context.cancelCtx"),
			networkIdentifier,
			&types.PartialBlockIdentifier{Index: &b.BlockIdentifier.Index},
		).Return(
			b,
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
		mockHandler.On(
			"BlockAdded",
			mock.AnythingOfType("*context.cancelCtx"),
			b,
		).Return(
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
	}

	err := syncer.Sync(ctx, 100, 1200)
	assert.NoError(t, err)
	mockHelper.AssertNumberOfCalls(t, "NetworkStatus", 3)
	mockHelper.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func TestSync_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockHelper := &mocks.Helper{}
	mockHandler := &mocks.Handler{}
	syncer := New(networkIdentifier, mockHelper, mockHandler, cancel, WithConcurrency(16))

	// Force syncer to only get part of the way through the full range
	mockHelper.On("NetworkStatus", ctx, networkIdentifier).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 200",
			Index: 200,
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 0",
			Index: 0,
		},
	}, nil).Twice()

	mockHelper.On("NetworkStatus", ctx, networkIdentifier).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 1300",
			Index: 1300,
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 0",
			Index: 0,
		},
	}, nil).Twice()

	blocks := createBlocks(0, 1200, "")
	for _, b := range blocks {
		mockHelper.On(
			"Block",
			mock.AnythingOfType("*context.cancelCtx"),
			networkIdentifier,
			&types.PartialBlockIdentifier{Index: &b.BlockIdentifier.Index},
		).Return(
			b,
			nil,
		).Once()
		mockHandler.On(
			"BlockAdded",
			mock.AnythingOfType("*context.cancelCtx"),
			b,
		).Return(
			nil,
		).Once()
	}

	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()

	err := syncer.Sync(ctx, -1, 1200)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestSync_Reorg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockHelper := &mocks.Helper{}
	mockHandler := &mocks.Handler{}
	syncer := New(networkIdentifier, mockHelper, mockHandler, cancel, WithConcurrency(16))

	mockHelper.On("NetworkStatus", ctx, networkIdentifier).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 1300",
			Index: 1300,
		},
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Hash:  "block 0",
			Index: 0,
		},
	}, nil).Run(func(args mock.Arguments) {
		err := args.Get(0).(context.Context)
		assert.NoError(t, err.Err())
	})

	blocks := createBlocks(0, 800, "")
	for _, b := range blocks { // [0, 800]
		mockHelper.On(
			"Block",
			mock.AnythingOfType("*context.cancelCtx"),
			networkIdentifier,
			&types.PartialBlockIdentifier{Index: &b.BlockIdentifier.Index},
		).Return(
			b,
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
		mockHandler.On(
			"BlockAdded",
			mock.AnythingOfType("*context.cancelCtx"),
			b,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Return(
			nil,
		).Once()
	}

	// Create reorg
	newBlocks := createBlocks(790, 1200, "other")
	mockHelper.On(
		"Block",
		mock.AnythingOfType("*context.cancelCtx"),
		networkIdentifier,
		&types.PartialBlockIdentifier{Index: &newBlocks[11].BlockIdentifier.Index},
	).Return(
		newBlocks[11],
		nil,
	).Run(func(args mock.Arguments) {
		err := args.Get(0).(context.Context)
		assert.NoError(t, err.Err())
	}).Once() // [801]

	// Set parent of reorg start to be last good block
	newBlocks[0].ParentBlockIdentifier = blocks[789].BlockIdentifier

	// Orphan last 10 blocks
	for i := 790; i <= 800; i++ { // [790, 800]
		thisBlock := newBlocks[i-790]
		mockHelper.On(
			"Block",
			mock.AnythingOfType("*context.cancelCtx"),
			networkIdentifier,
			&types.PartialBlockIdentifier{Index: &thisBlock.BlockIdentifier.Index},
		).Return(
			thisBlock,
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
		mockHandler.On(
			"BlockRemoved",
			mock.AnythingOfType("*context.cancelCtx"),
			blocks[i].BlockIdentifier,
		).Return(
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
	}

	mockHandler.On(
		"BlockAdded",
		mock.AnythingOfType("*context.cancelCtx"),
		newBlocks[0],
	).Run(func(args mock.Arguments) {
		err := args.Get(0).(context.Context)
		assert.NoError(t, err.Err())
	}).Return(
		nil,
	).Once() // only fetch this block once

	// New blocks added
	for _, b := range newBlocks[1:] { // [790, 1200]
		mockHelper.On(
			"Block",
			mock.AnythingOfType("*context.cancelCtx"),
			networkIdentifier,
			&types.PartialBlockIdentifier{Index: &b.BlockIdentifier.Index},
		).Return(
			b,
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
		mockHandler.On(
			"BlockAdded",
			mock.AnythingOfType("*context.cancelCtx"),
			b,
		).Return(
			nil,
		).Run(func(args mock.Arguments) {
			assertNotCanceled(t, args)
		}).Once()
	}

	// Expected Calls to Block
	// [0, 789] = 1
	// [790] = 2
	// [791, 800] = 3
	// [801] = 2
	// [802,1200] = 1

	err := syncer.Sync(ctx, -1, 1200)
	assert.NoError(t, err)
	mockHelper.AssertNumberOfCalls(t, "NetworkStatus", 3)
	mockHelper.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}
