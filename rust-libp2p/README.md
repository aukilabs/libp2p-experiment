```
wasm-pack build --target web

npm install ../../rust-libp2p/pkg
```

## Invalid version of libp2p-webrtc
https://crates.io/crates/libp2p-webrtc/versions

## Invalid version of libp2p-webrtc-websys
https://github.com/libp2p/rust-libp2p/pull/5569, https://doc.rust-lang.org/cargo/reference/specifying-dependencies.html#choice-of-commit

## Can't resolve `env`
https://github.com/rustwasm/wasm-bindgen/discussions/3500
https://github.com/rustwasm/wasm-bindgen/issues/2160
https://webassembly.github.io/wabt/demo/wasm2wat/ or install from https://github.com/WebAssembly/wabt/releases
```

$ ~/Downloads/wabt-1.0.36/bin/wasm2wat pkg/rust_libp2p_bg.wasm | grep env
  (import "env" "now" (func (;43;) (type 27)))

$ ../find_wasm_import/target/debug/find_wasm_import env now rust-libp2p/target/wasm32-unknown-unknown/release/deps 
"env" "now" is imported by:
  rust-libp2p/target/wasm32-unknown-unknown/release/deps/libinstant-8969c24ef802a1e3.rlib
  rust-libp2p/target/wasm32-unknown-unknown/release/deps/libinstant-2a5cdcf256dd3f2d.rlib
```
