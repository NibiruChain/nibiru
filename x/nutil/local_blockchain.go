package nutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

const localBlockchainStatusURL = "http://localhost:26657/status"

// EnsureLocalBlockchain verifies that a local nibid process is running and
// serving a healthy CometBFT status endpoint.
func EnsureLocalBlockchain() error {
	cmd := exec.Command("pgrep", "-x", "nibid")
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) == 0 {
		return fmt.Errorf(`failed to run "pgrep -x nibid" and parse PID; local network is not online: %w`, err)
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(localBlockchainStatusURL)
	if err != nil {
		return fmt.Errorf("failed to reach nibid at %s: %w", localBlockchainStatusURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from nibid: %d", resp.StatusCode)
	}

	var data struct {
		Result struct {
			NodeInfo struct {
				Network string `json:"network"`
				ID      string `json:"id"`
			} `json:"node_info"`
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode /status response: %w", err)
	}

	return nil
}
