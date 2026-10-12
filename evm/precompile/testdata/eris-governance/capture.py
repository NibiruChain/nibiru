#!/usr/bin/env python3
"""Capture public mainnet contract code and storage for governance replay. Never signs."""
import base64
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

NODE = "https://rpc.nibiru.fi:443"
ADDRESSES = {
    "hub": "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m",
    "escrow": "nibi1us4rh4a9rexvde8l3m8f8nlz6wcf9qg57zk2w06kqkm4vtp96vkst9kuwd",
    "gauge": "nibi1qh59hdelxfwah7g7e8k0lxu4upqatq5w4jdw60yv9shhlhm86ckq4txl92",
    "proposal_gauge": "nibi19g4zl2rac0ljtxrwqrd0lqgaca7ettn3p2udu4w3zamk84fae72ssvf2le",
}
OUT = Path(__file__).resolve().parent
status = json.loads(subprocess.check_output(["nibid", "status", "--node", NODE]))
sync = status.get("SyncInfo", status.get("sync_info"))
height = int(sync["latest_block_height"])

def query(*parts):
    return json.loads(subprocess.check_output(["nibid", "q", *parts, "--node", NODE, "--height", str(height), "--output", "json"], stderr=subprocess.PIPE))

block = json.loads(subprocess.check_output(["nibid", "q", "block", str(height), "--node", NODE]))
header = block["block"]["header"]
assert header["chain_id"] == "cataclysm-1"
result = {"height": height, "time": header["time"], "node": NODE, "contracts": []}
for name, address in ADDRESSES.items():
    contract = query("wasm", "contract", address)
    code_id = contract["contract_info"]["code_id"]
    code = query("wasm", "code-info", str(code_id))
    entries, page_key = [], None
    while True:
        args = ["wasm", "contract-state", "all", address, "--limit", "1000"]
        if page_key: args += ["--page-key", page_key]
        page = query(*args)
        entries += page["models"]
        page_key = page["pagination"]["next_key"]
        if not page_key: break
    with tempfile.TemporaryDirectory() as tmp:
        path = Path(tmp) / "contract.wasm"
        subprocess.run(["nibid", "q", "wasm", "code", str(code_id), str(path), "--node", NODE, "--height", str(height)], check=True, capture_output=True)
        bytecode = path.read_bytes()
        digest = hashlib.sha256(bytecode).hexdigest()
        assert digest == code["data_hash"].lower()
        (OUT / f"{name}.wasm.gz").write_bytes(gzip.compress(bytecode, mtime=0))
    result["contracts"].append({"name": name, "address": address, "info": contract["contract_info"], "hash": digest, "models": entries, "history": query("wasm", "contract-history", address)["entries"]})
    print(f"{name}: {len(entries)} entries, {len(bytecode)} bytes", flush=True)
(OUT / "snapshot.json").write_text(json.dumps(result, indent=2) + "\n")
print(f"Captured mainnet block {height}", flush=True)
