// Package abis embeds contract ABI JSON files at compile time.
package abis

import _ "embed"

//go:embed PerpVaultEvmInterface.json
var PerpVaultEvmInterface []byte
