// Copyright 2021 Coinbase, Inc.
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

// Generated by: OpenAPI Generator (https://openapi-generator.tech)

package types

// Operation Operations contain all balance-changing information within a transaction. They are
// always one-sided (only affect 1 AccountIdentifier) and can succeed or fail independently from a
// Transaction. Operations are used both to represent on-chain data (Data API) and to construct new
// transactions (Construction API), creating a standard interface for reading and writing to
// blockchains.
type Operation struct {
	OperationIdentifier *OperationIdentifier `json:"operation_identifier"`
	// Restrict referenced related_operations to identifier indices < the current
	// operation_identifier.index. This ensures there exists a clear DAG-structure of relations.
	// Since operations are one-sided, one could imagine relating operations in a single transfer or
	// linking operations in a call tree.
	RelatedOperations []*OperationIdentifier `json:"related_operations,omitempty"`
	// Type is the network-specific type of the operation. Ensure that any type that can be returned
	// here is also specified in the NetworkOptionsResponse. This can be very useful to downstream
	// consumers that parse all block data.
	Type string `json:"type"`
	// Status is the network-specific status of the operation. Status is not defined on the
	// transaction object because blockchains with smart contracts may have transactions that
	// partially apply (some operations are successful and some are not). Blockchains with atomic
	// transactions (all operations succeed or all operations fail) will have the same status for
	// each operation. On-chain operations (operations retrieved in the `/block` and
	// `/block/transaction` endpoints) MUST have a populated status field (anything on-chain must
	// have succeeded or failed). However, operations provided during transaction construction
	// (often times called \"intent\" in the documentation) MUST NOT have a populated status field
	// (operations yet to be included on-chain have not yet succeeded or failed).
	Status     *string                `json:"status,omitempty"`
	Account    *AccountIdentifier     `json:"account,omitempty"`
	Amount     *Amount                `json:"amount,omitempty"`
	CoinChange *CoinChange            `json:"coin_change,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
