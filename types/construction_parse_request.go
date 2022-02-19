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

// Generated by: OpenAPI Generator (https://openapi-generator.tech)

package types

// ConstructionParseRequest ConstructionParseRequest is the input to the `/construction/parse`
// endpoint. It allows the caller to parse either an unsigned or signed transaction.
type ConstructionParseRequest struct {
	NetworkIdentifier *NetworkIdentifier `json:"network_identifier"`
	// Signed is a boolean indicating whether the transaction is signed.
	Signed bool `json:"signed"`
	// This must be either the unsigned transaction blob returned by `/construction/payloads` or the
	// signed transaction blob returned by `/construction/combine`.
	Transaction string `json:"transaction"`
}
