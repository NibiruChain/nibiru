package keeper_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// saiPulseFixtures loads the checked-in Wasm once per test process, including
// concurrent callers. The shared byte slices are immutable. This cache and its
// loader are compiled only into the test binary.
var saiPulseFixtures = sync.OnceValues(func() (map[string][]byte, error) {
	fixtures := make(map[string][]byte, 3)
	for _, name := range []string{"sai_pulse.wasm", "vault.wasm", "vault_token_minter.wasm"} {
		code, err := os.ReadFile(filepath.Join("..", "testdata", "sai-pulse", name))
		if err != nil {
			return nil, fmt.Errorf("load Sai Pulse fixture %s: %w", name, err)
		}
		fixtures[name] = code
	}
	return fixtures, nil
})
