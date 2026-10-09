#!/usr/bin/env python3
"""Capture public Eris recovery inputs at a single mainnet height. Never signs."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

ERIS = 'nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m'
ATTACKER = 'nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8'
CW3 = 'nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v'
EXPECTED_HASH = 'e1c2be31ae008a015efa16b51f74ca1a014f4b1d0952da12ba3f60aaffa66321'

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--node', default='https://rpc.nibiru.fi:443')
parser.add_argument('--height', type=int)
args = parser.parse_args()
out = Path(__file__).resolve().parent


def query(*parts):
    command = ['nibid', 'q', *parts, '--node', args.node]
    if parts[0] != 'block':
        command.extend(['--height', str(height), '--output', 'json'])
    return json.loads(subprocess.check_output(command, text=True))


status = json.loads(subprocess.check_output(['nibid', 'status', '--node', args.node], text=True))
height = args.height or int(status['SyncInfo']['latest_block_height'])
block = query('block', str(height))
header = block['block']['header']
assert header['chain_id'] == 'cataclysm-1'
contract = query('wasm', 'contract', ERIS)
code_id = contract['contract_info']['code_id']
code = query('wasm', 'code-info', str(code_id))
assert code['data_hash'].lower() == EXPECTED_HASH, code
models = []
page_key = None
while True:
    parts = ['wasm', 'contract-state', 'all', ERIS, '--limit', '100']
    if page_key:
        parts.extend(['--page-key', page_key])
    page = query(*parts)
    models.extend(page['models'])
    page_key = page['pagination']['next_key']
    if not page_key:
        break

with tempfile.TemporaryDirectory() as tmp:
    bytecode_file = Path(tmp) / 'hub.wasm'
    subprocess.run(['nibid', 'q', 'wasm', 'code', str(code_id), str(bytecode_file),
                    '--node', args.node, '--height', str(height)], check=True, capture_output=True)
    bytecode = bytecode_file.read_bytes()
    assert hashlib.sha256(bytecode).hexdigest() == EXPECTED_HASH
    (out / 'hub.wasm.gz').write_bytes(gzip.compress(bytecode, mtime=0))

snapshot = dict(
    height=height, time=header['time'], block_hash=block['block_id']['hash'], node=args.node,
    contract=contract, code=code,
    history=query('wasm', 'contract-history', ERIS)['entries'], models=models,
    cw3=query('wasm', 'contract', CW3),
    balances={a: query('bank', 'balances', a, '--denom', 'unibi') for a in [ERIS, ATTACKER, CW3]},
    requests=query('wasm', 'contract-state', 'smart', ERIS,
                   json.dumps({'unbond_requests_by_user_details': {'user': ATTACKER, 'limit': 2}}))['data'],
)
(out / 'snapshot.json').write_text(json.dumps(snapshot, indent=2) + '\n')
print(f'Captured height {height}, {len(models)} storage entries, {len(bytecode)} Wasm bytes')
