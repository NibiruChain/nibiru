# Bundled WasmVM libraries

The shared libraries come from [lib/wasmvm/v1.13.2](https://github.com/NibiruChain/nibiru/releases/tag/lib%2Fwasmvm%2Fv1.13.2).
The release uses Wasmer 7.4.2 and reports WasmVM ABI version
`1.5.10-nibiru.1`. It adds saturating gas accounting, complete static
validation of uploaded contracts, and freeing of out-of-gas error buffers.

The artifacts were built and tested by [tag CI run 40265684557](https://github.com/NibiruChain/nibiru/actions/runs/40265684557)
at release source commit `2618092ba4df81092246e34913d26680ca711bf8` on main.
All six library assets and the checksum manifest were verified against their
GitHub SHA-256 digests before updating the bundled libraries.
Their SHA-256 digests are pinned in `contrib/scripts/wasmvm-checksums.txt`.
