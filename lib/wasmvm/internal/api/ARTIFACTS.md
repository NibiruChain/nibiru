# Bundled WasmVM libraries

The shared libraries come from [lib/wasmvm/v1.13.0](https://github.com/NibiruChain/nibiru/releases/tag/lib%2Fwasmvm%2Fv1.13.0).
The release contains the Wasmer 7.4.2 fixes and reports WasmVM ABI version
`1.5.10-nibiru.1`.

The artifacts were built and tested by [CI run 37596918070](https://github.com/NibiruChain/nibiru/actions/runs/37596918070)
at commit `8f3eeb5d2171a3b855d2b51bfb7e01545e31a68b`. The runtime sources are
identical to release commit `6faa367137af757dec8d93c7fa4802f3e414881f` on main.
Their SHA-256 digests are pinned in `contrib/scripts/wasmvm-checksums.txt`.
