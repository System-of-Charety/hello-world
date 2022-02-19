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
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// PopulateInput populates user defined variables in the input
// with their corresponding values from the execution state.
func PopulateInput(state string, input string) (string, error) {
	re := regexp.MustCompile(`\{\{[^\}]*\}\}`)

	var err error
	input = re.ReplaceAllStringFunc(input, func(match string) string {
		// remove special characters
		match = strings.Replace(match, "{{", "", 1)
		match = strings.Replace(match, "}}", "", 1)

		value := gjson.Get(state, match)
		if !value.Exists() {
			err = fmt.Errorf("%s is not present in state", match)
			return ""
		}

		return value.Raw
	})
	if err != nil {
		return "", fmt.Errorf("%w: unable to insert variables", err)
	}

	if !gjson.Valid(input) {
		return "", errors.New("populated input is not valid JSON")
	}

	return input, nil
}
