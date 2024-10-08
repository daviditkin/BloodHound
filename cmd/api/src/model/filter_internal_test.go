// Copyright 2023 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryParameterFilterParser_ParseQueryParameterFilter(t *testing.T) {
	parser := NewQueryParameterFilterParser()

	t.Run("parser should parse a parameter filter", func(t *testing.T) {
		if filter, err := parser.ParseQueryParameterFilter("parameter", "eq:auth.value"); err != nil {
			t.Fatalf("Failed parsing query parameter: %v", err)
		} else {
			require.Equal(t, filter.Name, "parameter")
			require.Equal(t, "auth.value", filter.Value)
		}
	})

	t.Run("parser should parse a parameter with ~", func(t *testing.T) {
		if filter, err := parser.ParseQueryParameterFilter("parameter", "~eq:auth.value"); err != nil {
			t.Fatalf("Failed parsing query parameter: %v", err)
		} else {
			require.Equal(t, filter.Name, "parameter")
			require.Equal(t, "auth.value", filter.Value)
		}
	})

	t.Run("parser should parse a parameter filter with spacing", func(t *testing.T) {
		if filter, err := parser.ParseQueryParameterFilter("parameter", "eq:hello world"); err != nil {
			t.Fatalf("Failed parsing query parameter: %v", err)
		} else {
			require.Equal(t, filter.Name, "parameter")
			require.Equal(t, "hello world", filter.Value)
		}
	})

	t.Run("error when parsing an invalid parameter", func(t *testing.T) {
		_, err := parser.ParseQueryParameterFilter("parameter", "eq : hello world")
		require.Error(t, err)
	})

}
