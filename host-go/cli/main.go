// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/sourcenetwork/immutable/enumerable"
	"github.com/sourcenetwork/lens/host-go/config"
)

func main() {
	lensFilePath := os.Args[1]

	dataBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var data []map[string]any
	err = json.Unmarshal(dataBytes, &data)
	if err != nil {
		panic(err)
	}

	src := enumerable.New(data)
	result, err := config.LoadFromFile[map[string]any, map[string]any](lensFilePath, src)
	if err != nil {
		panic(err)
	}

	results := []any{}
	for {
		hasNext, err := result.Next()
		if err != nil {
			panic(err)
		}

		if !hasNext {
			break
		}

		val, err := result.Value()
		if err != nil {
			panic(err)
		}

		results = append(results, val)
	}

	resultJson, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}

	os.Stdout.WriteString(string(resultJson))
}
