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

// ConstructionPreprocessRequest ConstructionPreprocessRequest is passed to the
// `/construction/preprocess` endpoint so that a Rosetta implementation can determine which metadata
// it needs to request for construction.
type ConstructionPreprocessRequest struct {
	NetworkIdentifier *NetworkIdentifier     `json:"network_identifier"`
	Operations        []*Operation           `json:"operations"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}
