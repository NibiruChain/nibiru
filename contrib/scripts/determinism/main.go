// Command determinism drives `just chaos determinism` (see
// spec/02-chaosnet.md). It has two subcommands:
//
//	evm-load  Send signed EVM transactions to every node's JSON-RPC.
//	compare   Wait for every node to reach a target height, then compare
//	          /block and /block_results for every height across nodes.
//
// compare exits 1 and prints the first height where nodes disagree, or the
// height where a node stopped and how its app hash differs.
package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EvmLoadKeys are public test keys; never use them with real funds. The
// genesis funds their addresses (EVM_LOAD_ADDRS in
// contrib/docker-compose/docker-compose-chaosnet.yml, in the same order).
// Key i sends to node i, so each sender has one nonce sequence.
var EvmLoadKeys = []string{
	"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
	"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
	"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804e8ab365a",
	"7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
}

// counterInitCode deploys a contract whose runtime code adds the first
// calldata word to storage slot 0: PUSH1 0 CALLDATALOAD PUSH1 0 SLOAD ADD
// PUSH1 0 SSTORE STOP.
const counterInitCode = "600b600c600039600b6000f3" + "6000356000540160005500"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "evm-load":
		err = runEvmLoad(os.Args[2:])
	case "compare":
		err = runCompare(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "determinism: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: determinism evm-load|compare [flags]")
	os.Exit(2)
}

// ---------------------------------------------------------------------------
// evm-load
// ---------------------------------------------------------------------------

func runEvmLoad(args []string) error {
	fs := flag.NewFlagSet("evm-load", flag.ExitOnError)
	rpcs := fs.String("rpc", "", "comma-separated JSON-RPC URLs, one per node")
	duration := fs.Duration("duration", 60*time.Second, "how long to send")
	interval := fs.Duration("interval", 500*time.Millisecond, "pause per tx")
	_ = fs.Parse(args)

	urls := splitList(*rpcs)
	if len(urls) == 0 || len(urls) > len(EvmLoadKeys) {
		return fmt.Errorf("need 1 to %d -rpc URLs", len(EvmLoadKeys))
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var wg sync.WaitGroup
	stats := make([]evmStats, len(urls))
	for i, url := range urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			stats[i] = evmSender(ctx, url, EvmLoadKeys[i], *interval)
		}(i, url)
	}
	wg.Wait()

	for i, s := range stats {
		fmt.Printf("evm-load node %d (%s): sent %d, rejected %d\n",
			i, urls[i], s.sent, s.rejected)
	}
	return nil
}

type evmStats struct {
	sent     int
	rejected int
}

// evmSender sends a deploy, then alternates value transfers and counter
// calls until ctx ends. Errors are counted, not fatal: nodes restart while
// this runs.
func evmSender(
	ctx context.Context, url string, hexKey string, interval time.Duration,
) (stats evmStats) {
	key, err := gethcrypto.HexToECDSA(hexKey)
	if err != nil {
		panic(err)
	}
	from := gethcrypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("evm-load: %s sends to %s\n", from.Hex(), url)

	var (
		client   *ethclient.Client
		chainID  *big.Int
		nonce    uint64
		synced   bool
		contract *gethcommon.Address
	)
	for i := 0; ctx.Err() == nil; i++ {
		if !sleepCtx(ctx, interval) {
			break
		}
		if client == nil {
			client, err = ethclient.DialContext(ctx, url)
			if err != nil {
				client = nil
				continue
			}
		}
		if !synced {
			chainID, err = client.ChainID(ctx)
			if err != nil {
				client.Close()
				client = nil
				continue
			}
			nonce, err = client.PendingNonceAt(ctx, from)
			if err != nil {
				client.Close()
				client = nil
				continue
			}
			synced = true
		}

		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			synced = false
			continue
		}

		var txData gethcore.LegacyTx
		switch {
		case contract == nil:
			txData = gethcore.LegacyTx{
				Nonce: nonce, GasPrice: gasPrice, Gas: 200_000,
				Data: gethcommon.FromHex(counterInitCode),
			}
		case i%2 == 0:
			to := gethcommon.BigToAddress(big.NewInt(int64(0x1000 + i%7)))
			txData = gethcore.LegacyTx{
				Nonce: nonce, GasPrice: gasPrice, Gas: 21_000,
				To: &to, Value: big.NewInt(1_000_000_000_000),
			}
		default:
			txData = gethcore.LegacyTx{
				Nonce: nonce, GasPrice: gasPrice, Gas: 100_000,
				To:   contract,
				Data: gethcommon.LeftPadBytes(big.NewInt(int64(i)).Bytes(), 32),
			}
		}

		if err := sendSigned(ctx, client, key, chainID, &txData); err != nil {
			stats.rejected++
			synced = false // re-read the nonce
			continue
		}
		if contract == nil {
			addr := gethcrypto.CreateAddress(from, nonce)
			contract = &addr
		}
		nonce++
		stats.sent++
	}
	if client != nil {
		client.Close()
	}
	return stats
}

func sendSigned(
	ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey,
	chainID *big.Int, txData *gethcore.LegacyTx,
) error {
	tx, err := gethcore.SignTx(
		gethcore.NewTx(txData), gethcore.NewEIP155Signer(chainID), key,
	)
	if err != nil {
		return err
	}
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return client.SendTransaction(sendCtx, tx)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// ---------------------------------------------------------------------------
// compare
// ---------------------------------------------------------------------------

// Node is one Nibiru BFT RPC endpoint.
type Node struct {
	Name string
	URL  string
}

func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	rpcs := fs.String("rpc", "", "comma-separated Nibiru BFT RPC URLs")
	names := fs.String("names", "", "comma-separated node names")
	target := fs.Int64("target", 0, "height every node must reach")
	stall := fs.Duration("stall", 60*time.Second,
		"fail if a node's height does not move for this long")
	timeout := fs.Duration("timeout", 15*time.Minute, "overall wait limit")
	failedOut := fs.String("failed-out", "",
		"write names of nodes that stopped to this file, one per line")
	_ = fs.Parse(args)

	nodes, err := parseNodes(*rpcs, *names)
	if err != nil {
		return err
	}
	if *target < 1 {
		return errors.New("-target must be at least 1")
	}

	heights, stopped := waitForHeight(nodes, *target, *stall, *timeout)
	if len(stopped) > 0 {
		if *failedOut != "" {
			var b strings.Builder
			for _, n := range stopped {
				b.WriteString(n.Name + "\n")
			}
			_ = os.WriteFile(*failedOut, []byte(b.String()), 0o644)
		}
		reportStopped(nodes, heights, stopped)
		return fmt.Errorf("%d node(s) stopped before height %d",
			len(stopped), *target)
	}

	fmt.Printf("compare: all %d nodes reached height %d; comparing 1..%d\n",
		len(nodes), *target, *target)
	var txs, failedTxs int
	for h := int64(1); h <= *target; h++ {
		view, err := compareHeight(nodes, h)
		if err != nil {
			return err
		}
		txs += len(view.TxResults)
		for _, r := range view.TxResults {
			if r.Code != 0 {
				failedTxs++
			}
		}
	}
	fmt.Printf("compare: OK, %d nodes agree on heights 1..%d "+
		"(%d txs, %d with a non-zero code)\n",
		len(nodes), *target, txs, failedTxs)
	return nil
}

func parseNodes(rpcs, names string) ([]Node, error) {
	urls, ns := splitList(rpcs), splitList(names)
	if len(urls) == 0 {
		return nil, errors.New("-rpc is required")
	}
	if len(ns) != 0 && len(ns) != len(urls) {
		return nil, errors.New("-names must match -rpc in length")
	}
	nodes := make([]Node, len(urls))
	for i, u := range urls {
		name := fmt.Sprintf("node-%d", i)
		if len(ns) != 0 {
			name = ns[i]
		}
		nodes[i] = Node{Name: name, URL: strings.TrimRight(u, "/")}
	}
	return nodes, nil
}

// waitForHeight polls /status until every node reaches target. It returns
// the last height seen per node, and the nodes that did not move for the
// stall duration or were still short of target at the timeout.
func waitForHeight(
	nodes []Node, target int64, stall, timeout time.Duration,
) (heights []int64, stopped []Node) {
	heights = make([]int64, len(nodes))
	lastMove := make([]time.Time, len(nodes))
	start := time.Now()
	for i := range lastMove {
		lastMove[i] = start
	}
	lastLog := time.Time{}
	for {
		done := true
		stopped = stopped[:0]
		for i, n := range nodes {
			h, err := latestHeight(n)
			if err == nil && h > heights[i] {
				heights[i] = h
				lastMove[i] = time.Now()
			}
			if heights[i] < target {
				done = false
				if time.Since(lastMove[i]) > stall {
					stopped = append(stopped, n)
				}
			}
		}
		if done {
			return heights, nil
		}
		if len(stopped) > 0 || time.Since(start) > timeout {
			if len(stopped) == 0 {
				for i, n := range nodes {
					if heights[i] < target {
						stopped = append(stopped, n)
					}
				}
			}
			return heights, stopped
		}
		if time.Since(lastLog) > 10*time.Second {
			fmt.Printf("compare: waiting for height %d: %v\n", target, heights)
			lastLog = time.Now()
		}
		time.Sleep(time.Second)
	}
}

// reportStopped explains why nodes stopped. For each stopped node it compares
// the app hash the node computed after its last height (from /abci_info) with
// the app hash in the next block's header on another node, or with the other
// nodes' app hashes at the same height when the whole network halted.
func reportStopped(nodes []Node, heights []int64, stopped []Node) {
	fmt.Println("compare: FAIL: node(s) stopped before the target height")
	infos := make([]abciInfo, len(nodes))
	for i, n := range nodes {
		info, err := getABCIInfo(n)
		if err != nil {
			fmt.Printf("  %s: unreachable (%v), last seen height %d\n",
				n.Name, err, heights[i])
			continue
		}
		infos[i] = info
		fmt.Printf("  %s: height %d, app hash %s\n",
			n.Name, info.Height, info.AppHash)
	}

	firstDiff := int64(-1)
	for i, n := range nodes {
		if infos[i].Height == 0 || !containsNode(stopped, n) {
			continue
		}
		h := infos[i].Height
		// Another node with block h+1 knows the agreed app hash after h.
		for j, other := range nodes {
			if j == i || heights[j] <= h {
				continue
			}
			b, err := getBlock(other, h+1)
			if err != nil {
				continue
			}
			if b.AppHash != infos[i].AppHash {
				fmt.Printf("  %s: app hash after height %d is %s, but block "+
					"%d on %s carries %s\n", n.Name, h, infos[i].AppHash,
					h+1, other.Name, b.AppHash)
				if firstDiff < 0 || h < firstDiff {
					firstDiff = h
				}
			}
			break
		}
	}

	// Nodes stuck at the same height with different app hashes.
	byHeight := map[int64]map[string][]string{}
	for i, n := range nodes {
		if infos[i].Height == 0 {
			continue
		}
		h := infos[i].Height
		if byHeight[h] == nil {
			byHeight[h] = map[string][]string{}
		}
		byHeight[h][infos[i].AppHash] = append(byHeight[h][infos[i].AppHash], n.Name)
	}
	for h, hashes := range byHeight {
		if len(hashes) < 2 {
			continue
		}
		fmt.Printf("  nodes at height %d disagree on the app hash:\n", h)
		for hash, ns := range hashes {
			fmt.Printf("    %s: %s\n", hash, strings.Join(ns, ", "))
		}
		if firstDiff < 0 || h < firstDiff {
			firstDiff = h
		}
	}

	if firstDiff >= 0 {
		fmt.Printf("compare: first height where app hashes differ: %d\n",
			firstDiff)
	} else {
		fmt.Println("compare: no app hash difference found; see node logs")
	}
}

func containsNode(ns []Node, n Node) bool {
	for _, m := range ns {
		if m.Name == n.Name {
			return true
		}
	}
	return false
}

// heightView holds the fields compared for one height on one node.
type heightView struct {
	BlockID         string
	AppHash         string
	LastResultsHash string
	TxResults       []txResult
}

type txResult struct {
	Code    uint32
	GasUsed string
}

// compareHeight compares height h across nodes and returns the first node's
// view when they agree.
func compareHeight(nodes []Node, h int64) (heightView, error) {
	views := make([]heightView, len(nodes))
	errs := make([]error, len(nodes))
	var wg sync.WaitGroup
	for i, n := range nodes {
		wg.Add(1)
		go func(i int, n Node) {
			defer wg.Done()
			views[i], errs[i] = getHeightView(n, h)
		}(i, n)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return heightView{}, fmt.Errorf("height %d on %s: %w", h, nodes[i].Name, err)
		}
	}

	base := views[0]
	for i := 1; i < len(views); i++ {
		v := views[i]
		field, a, b := diffView(base, v)
		if field == "" {
			continue
		}
		fmt.Printf("compare: FAIL: first difference at height %d, field %s\n",
			h, field)
		fmt.Printf("  %s: %s\n  %s: %s\n", nodes[0].Name, a, nodes[i].Name, b)
		return heightView{}, fmt.Errorf("nodes disagree at height %d (%s)", h, field)
	}
	return base, nil
}

// diffView returns the first differing field and both values, or "".
func diffView(a, b heightView) (field, va, vb string) {
	switch {
	case a.BlockID != b.BlockID:
		return "block_id.hash", a.BlockID, b.BlockID
	case a.AppHash != b.AppHash:
		return "header.app_hash", a.AppHash, b.AppHash
	case a.LastResultsHash != b.LastResultsHash:
		return "header.last_results_hash", a.LastResultsHash, b.LastResultsHash
	case len(a.TxResults) != len(b.TxResults):
		return "block_results.txs_results (count)",
			strconv.Itoa(len(a.TxResults)), strconv.Itoa(len(b.TxResults))
	}
	for k := range a.TxResults {
		ta, tb := a.TxResults[k], b.TxResults[k]
		if ta.Code != tb.Code {
			return fmt.Sprintf("block_results.txs_results[%d].code", k),
				strconv.Itoa(int(ta.Code)), strconv.Itoa(int(tb.Code))
		}
		if ta.GasUsed != tb.GasUsed {
			return fmt.Sprintf("block_results.txs_results[%d].gas_used", k),
				ta.GasUsed, tb.GasUsed
		}
	}
	return "", "", ""
}

// ---------------------------------------------------------------------------
// Nibiru BFT RPC
// ---------------------------------------------------------------------------

var httpClient = &http.Client{Timeout: 10 * time.Second}

func rpcGet(n Node, path string, out any) error {
	resp, err := httpClient.Get(n.URL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("GET %s: %s %s", path,
			envelope.Error.Message, envelope.Error.Data)
	}
	return json.Unmarshal(envelope.Result, out)
}

func latestHeight(n Node) (int64, error) {
	var res struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	}
	if err := rpcGet(n, "/status", &res); err != nil {
		return 0, err
	}
	return strconv.ParseInt(res.SyncInfo.LatestBlockHeight, 10, 64)
}

type abciInfo struct {
	Height  int64
	AppHash string
}

func getABCIInfo(n Node) (abciInfo, error) {
	var res struct {
		Response struct {
			LastBlockHeight  string `json:"last_block_height"`
			LastBlockAppHash string `json:"last_block_app_hash"`
		} `json:"response"`
	}
	if err := rpcGet(n, "/abci_info", &res); err != nil {
		return abciInfo{}, err
	}
	h, err := strconv.ParseInt(res.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return abciInfo{}, err
	}
	// last_block_app_hash is base64; headers print hex. Use hex for both.
	return abciInfo{Height: h, AppHash: b64ToHex(res.Response.LastBlockAppHash)}, nil
}

type blockFields struct {
	BlockID         string
	AppHash         string
	LastResultsHash string
}

func getBlock(n Node, h int64) (blockFields, error) {
	var res struct {
		BlockID struct {
			Hash string `json:"hash"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				AppHash         string `json:"app_hash"`
				LastResultsHash string `json:"last_results_hash"`
			} `json:"header"`
		} `json:"block"`
	}
	if err := rpcGet(n, fmt.Sprintf("/block?height=%d", h), &res); err != nil {
		return blockFields{}, err
	}
	return blockFields{
		BlockID:         res.BlockID.Hash,
		AppHash:         res.Block.Header.AppHash,
		LastResultsHash: res.Block.Header.LastResultsHash,
	}, nil
}

func getHeightView(n Node, h int64) (heightView, error) {
	b, err := getBlock(n, h)
	if err != nil {
		return heightView{}, err
	}
	var res struct {
		TxsResults []struct {
			Code    uint32 `json:"code"`
			GasUsed string `json:"gas_used"`
		} `json:"txs_results"`
	}
	if err := rpcGet(n, fmt.Sprintf("/block_results?height=%d", h), &res); err != nil {
		return heightView{}, err
	}
	v := heightView{
		BlockID: b.BlockID, AppHash: b.AppHash, LastResultsHash: b.LastResultsHash,
	}
	for _, r := range res.TxsResults {
		v.TxResults = append(v.TxResults, txResult{Code: r.Code, GasUsed: r.GasUsed})
	}
	return v, nil
}

// b64ToHex converts base64 to the upper-case hex that block headers use.
func b64ToHex(s string) string {
	bz, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return strings.ToUpper(hex.EncodeToString(bz))
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
