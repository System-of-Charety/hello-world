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

package constructor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/keys"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/tidwall/sjson"
)

// NewWorker returns a new Worker.
func NewWorker(helper WorkerHelper) *Worker {
	return &Worker{helper: helper}
}

func unmarshalInput(input []byte, output interface{}) error {
	// To prevent silent erroring, we explicitly
	// reject any unknown fields.
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&output); err != nil {
		return fmt.Errorf("%w: unable to unmarshal", err)
	}

	return nil
}

func (w *Worker) actions(ctx context.Context, state string, actions []*Action) (string, error) {
	for _, action := range actions {
		processedInput, err := PopulateInput(state, action.Input)
		if err != nil {
			return "", fmt.Errorf("%w: unable to populate variables", err)
		}

		var output string
		switch action.Type {
		case SetVariable:
			output = processedInput
		case GenerateKey:
			var unmarshaledInput GenerateKeyInput
			err = unmarshalInput([]byte(processedInput), &unmarshaledInput)
			if err != nil {
				return "", fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
			}

			output, err = GenerateKeyWorker(&unmarshaledInput)
		case Derive:
			var unmarshaledInput types.ConstructionDeriveRequest
			err = unmarshalInput([]byte(processedInput), &unmarshaledInput)
			if err != nil {
				return "", fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
			}

			output, err = w.DeriveWorker(ctx, &unmarshaledInput)
		default:
			return "", fmt.Errorf("%w: %s", ErrInvalidActionType, action.Type)
		}
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrActionFailed, err.Error())
		}

		// Update state at the specified output path.
		state, err = sjson.SetRaw(state, action.OutputPath, output)
		if err != nil {
			return "", fmt.Errorf("%w: unable to update state", err)
		}
	}

	return state, nil
}

// ProcessNextScenario performs the actions in the next available
// scenario.
func (w *Worker) ProcessNextScenario(
	ctx context.Context,
	j *Job,
) error {
	scenario := j.Scenarios[j.Index]
	newState, err := w.actions(ctx, j.State, scenario.Actions)
	if err != nil {
		return fmt.Errorf("%w: unable to process %s actions", err, scenario.Name)
	}

	j.State = newState
	j.Index++
	return nil
}

// DeriveWorker attempts to derive an address given a
// *types.ConstructionDeriveRequest input.
func (w *Worker) DeriveWorker(
	ctx context.Context,
	input *types.ConstructionDeriveRequest,
) (string, error) {
	address, metadata, err := w.helper.Derive(
		ctx,
		input.NetworkIdentifier,
		input.PublicKey,
		input.Metadata,
	)
	if err != nil {
		return "", err
	}

	return types.PrintStruct(types.ConstructionDeriveResponse{
		Address:  address,
		Metadata: metadata,
	}), nil
}

// GenerateKeyWorker attempts to generate a key given a
// *GenerateKeyInput input.
func GenerateKeyWorker(input *GenerateKeyInput) (string, error) {
	kp, err := keys.GenerateKeypair(input.CurveType)
	if err != nil {
		return "", err
	}

	return types.PrintStruct(kp), nil
}
