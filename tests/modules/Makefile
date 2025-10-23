.PHONY: deps\:build
deps\:build:
	rustup target add wasm32-unknown-unknown

.PHONY: build
build:
	$(MAKE) deps:build
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_simple/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --release --manifest-path "./rust_wasm32_simple/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_simple2/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_rename/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_counter/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_filter/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_normalize/Cargo.toml"
	cargo build --target wasm32-unknown-unknown --manifest-path "./rust_wasm32_memory/Cargo.toml"
	(cd "./as_wasm32_simple/" && npm install && npm run asbuild:debug)

.PHONY: build\:test
build\:test:
	cargo test --no-run --manifest-path "./rust_wasm32_simple/Cargo.toml"
	cargo test --no-run --manifest-path "./rust_wasm32_simple2/Cargo.toml"
	cargo test --no-run --manifest-path "./rust_wasm32_rename/Cargo.toml"
	cargo test --no-run --manifest-path "./rust_wasm32_counter/Cargo.toml"
	cargo test --no-run --manifest-path "./rust_wasm32_filter/Cargo.toml"
	cargo test --no-run --manifest-path "./rust_wasm32_normalize/Cargo.toml"
	cargo test --no-run --manifest-path "./rust_wasm32_memory/Cargo.toml"

.PHONY: test
test:
	cargo test --manifest-path "./rust_wasm32_simple/Cargo.toml"
	cargo test --manifest-path "./rust_wasm32_simple2/Cargo.toml"
	cargo test --manifest-path "./rust_wasm32_rename/Cargo.toml"
	cargo test --manifest-path "./rust_wasm32_counter/Cargo.toml"
	cargo test --manifest-path "./rust_wasm32_filter/Cargo.toml"
	cargo test --manifest-path "./rust_wasm32_normalize/Cargo.toml"
	cargo test --manifest-path "./rust_wasm32_memory/Cargo.toml"
