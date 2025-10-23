// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package modules

import (
	"path"
	"runtime"
)

// WasmPath1 contains a simple wasm32 rust lens that takes an item of `type1` and transforms it to `type2`.
var WasmPath1 string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_simple/target/wasm32-unknown-unknown/debug/rust_wasm32_simple.wasm",
)

// WasmPath1_Release contains a release build of `WasmPath1`, the binary size is considerably smaller (~180KB).
var WasmPath1_Release string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_simple/target/wasm32-unknown-unknown/release/rust_wasm32_simple.wasm",
)

// WasmPath2 contains a simple wasm32 rust lens that takes an item of `type2` and adds 1 to its age.
//
// Module also supplies an inverse function.
var WasmPath2 string = getPathRelativeToProjectRoot(
	"tests/modules/rust_wasm32_simple2/target/wasm32-unknown-unknown/debug/rust_wasm32_simple2.wasm",
)

// WasmPath3 contains a simple wasm32 assemblyScript lens that takes an item of `type2` and adds 10 to its age.
var WasmPath3 string = getPathRelativeToProjectRoot(
	"tests/modules/as_wasm32_simple/build/debug/as_wasm32_simple.wasm",
)

// WasmPath4 contains a wasm32 rust lens that takes two additional properties and a map and renames one of the properties.
var WasmPath4 string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_rename/target/wasm32-unknown-unknown/debug/rust_wasm32_rename.wasm",
)

// WasmPath5 contains a wasm32 rust lens that sets an id using a counter.
var WasmPath5 string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_counter/target/wasm32-unknown-unknown/debug/rust_wasm32_counter.wasm",
)

// WasmPath6 contains a wasm32 rust lens that only returns values where `__type` == "pass".
var WasmPath6 string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_filter/target/wasm32-unknown-unknown/debug/rust_wasm32_filter.wasm",
)

// WasmPath7 contains a wasm32 rust lens that turns books into multiple pages.
var WasmPath7 string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_normalize/target/wasm32-unknown-unknown/debug/rust_wasm32_normalize.wasm",
)

// WasmPath_Memory contains a wasm32 rust lens that copies the input item multiple times before returning it
// in order to check for memory related bugs.
var WasmPath_Memory string = getPathRelativeToProjectRoot(
	"/tests/modules/rust_wasm32_memory/target/wasm32-unknown-unknown/debug/rust_wasm32_memory.wasm",
)

func getPathRelativeToProjectRoot(relativePath string) string {
	_, filename, _, _ := runtime.Caller(0)
	root := path.Dir(path.Dir(path.Dir(filename)))
	return "file://" + path.Join(root, relativePath)
}
