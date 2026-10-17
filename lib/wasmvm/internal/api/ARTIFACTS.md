# Bundled WasmVM libraries

The shared libraries come from [lib/wasmvm/v1.13.1](https://github.com/NibiruChain/nibiru/releases/tag/lib%2Fwasmvm%2Fv1.13.1).
The release uses Wasmer 7.4.2 with the ELF frame-registration cleanup fix and
reports WasmVM ABI version `1.5.10-nibiru.1`.

The artifacts were built and tested by [tag CI run 37723722495](https://github.com/NibiruChain/nibiru/actions/runs/37723722495)
at release source commit `62d69cd58c7c9c3d7d6379d8dc2709561caacdbe` on main.
All six library assets and the checksum manifest were verified against their
GitHub SHA-256 digests before updating the bundled libraries.
Their SHA-256 digests are pinned in `contrib/scripts/wasmvm-checksums.txt`.
