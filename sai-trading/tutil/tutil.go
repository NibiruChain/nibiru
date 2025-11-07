package tutil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/NibiruChain/nibiru/v2/gosdk"
)

func EnsureBlockchain() error {
	cmd := exec.Command("pgrep", "-x", "nibid")
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) == 0 {
		return fmt.Errorf(`failed to run "pgrep -x nibid" and parse PID; local network is not online: %w`, err)
	}

	log.Printf("EnsureBlockchain: nibid process detected: PID(s): %q\n", string(out))

	// 2. Ping the /status endpoint
	netInfo := gosdk.NETWORK_INFO_DEFAULT
	url := fmt.Sprintf("%s/status", netInfo.TmRpcEndpoint)

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to reach nibid at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code from nibid: %d", resp.StatusCode)
	}

	// 3. Parse the JSON to confirm the node is healthy
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
