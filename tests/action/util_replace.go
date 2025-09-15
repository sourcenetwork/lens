// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package action

import (
	"bytes"
	"maps"
	"strconv"
	"strings"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/lens/tests/state"
)

// templateDataGenerators contains a set of data generators by their template prefix.
//
// Supporting action properties will replace any templated elements with data drawn from these
// sets.
var templateDataGenerators = map[string]func(*state.State) map[string]string{
	"LensIDs": func(s *state.State) map[string]string {
		res := map[string]string{}
		for i, ID := range s.LensIDs {
			res["LensIDs"+strconv.Itoa(i)] = ID.String()
		}

		return res
	},
	"WasmBytes": func(s *state.State) map[string]string {
		res := map[string]string{}
		for i, bytes := range s.WasmBytes {
			res["WasmBytes"+strconv.Itoa(i)] = "data://" + string(bytes)
		}

		return res
	},
}

// replace returns a new string with any templating placholders (see "text/template") with data drawn
// from `state`.
func replace(s *state.State, input string) string {
	if !strings.Contains(input, "{{") {
		// If the input doesn't contain any templating elements we can return early
		return input
	}

	templateData := map[string]string{}
	for _, datasetGenerator := range templateDataGenerators {
		// Having to regenerate the full dataset for every node-action is horribly inefficient, but
		// it is tolerable for now.
		maps.Copy(templateData, datasetGenerator(s))
	}

	tmpl := template.Must(template.New("").Parse(input))
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, templateData)
	require.NoError(s.T, err)

	return buf.String()
}
