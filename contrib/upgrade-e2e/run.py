#!/usr/bin/env python3
"""Run real governance upgrades without touching operator homes or global tools."""

from __future__ import annotations

import argparse
import contextlib
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import signal
import socket
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.parse
import urllib.request


REPO = Path(__file__).resolve().parents[2]
BASELINE_VERSION = "2.21.2"
BASELINE_CACHE = "v2.21.2--UkVfa3dET0d4ZEp2czRZYXptMw"
NEXT_UPGRADE = "e2e-x-upgrade-next"


class RunError(RuntimeError):
    """An acceptance assertion or bounded subprocess failed."""


def file_hash(path: Path) -> str:
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def extract_verified_binary(archive: Path, manifest: Path, output: Path) -> None:
    """Verify the release archive, then copy only its native executable bytes."""
    checksums = {}
    for line in manifest.read_text().splitlines():
        parts = line.split()
        if len(parts) == 2:
            checksums[parts[1].lstrip("*")] = parts[0]
    expected = checksums.get(archive.name)
    if not expected or file_hash(archive) != expected:
        raise RunError(f"Checksum verification failed: {archive}")
    with tarfile.open(archive, "r:gz") as bundle:
        matches = [m for m in bundle.getmembers() if m.isfile() and Path(m.name).name == "nibid"]
        if len(matches) != 1:
            raise RunError("Release archive must contain exactly one regular nibid file")
        with bundle.extractfile(matches[0]) as source, output.open("xb") as destination:
            shutil.copyfileobj(source, destination)
    output.chmod(0o755)


def set_toml_value(text: str, section: str, key: str, value: str) -> str:
    """Replace one existing scalar in its exact TOML section."""
    active = ""
    count = 0
    lines = []
    for line in text.splitlines(keepends=True):
        heading = re.fullmatch(r"\s*\[([^]]+)\]\s*", line.strip())
        if heading:
            active = heading.group(1)
        if active == section and re.match(rf"^\s*{re.escape(key)}\s*=", line):
            line = f"{key} = {value}\n"
            count += 1
        lines.append(line)
    if count != 1:
        raise RunError(f"Expected one TOML key [{section}] {key}; found {count}")
    return "".join(lines)


def require_unused_ports(ports: list[int]) -> None:
    with contextlib.ExitStack() as stack:
        for port in ports:
            sock = stack.enter_context(socket.socket())
            # A previous owned listener can leave TIME_WAIT connections after
            # shutdown. Match normal server reuse without permitting a second
            # listening socket on an occupied address.
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            try:
                sock.bind(("127.0.0.1", port))
            except OSError as error:
                raise RunError(f"Port {port} is already in use") from error


def find_cache() -> Path:
    local = REPO / "x/clidist/artifacts/releases" / BASELINE_CACHE
    if local.is_dir():
        return local
    result = subprocess.run(
        ["git", "rev-parse", "--git-common-dir"], cwd=REPO,
        check=True, text=True, capture_output=True,
    )
    common = Path(result.stdout.strip())
    if not common.is_absolute():
        common = REPO / common
    return common.resolve().parent / "x/clidist/artifacts/releases" / BASELINE_CACHE


class Runner:
    def __init__(self, args: argparse.Namespace):
        self.args = args
        self.root = args.run_dir
        if self.root:
            self.root = self.root.expanduser().resolve()
            self.root.mkdir(mode=0o700, parents=True, exist_ok=False)
        else:
            parent = REPO / ".cache/upgrade-e2e"
            parent.mkdir(parents=True, exist_ok=True)
            self.root = Path(tempfile.mkdtemp(prefix="run-", dir=parent))
        self.home = self.root / "node"
        self.artifacts = self.root / "artifacts"
        self.evidence = self.root / "evidence"
        self.artifacts.mkdir()
        self.evidence.mkdir()
        self.baseline = self.artifacts / "baseline-nibid"
        self.candidate = self.artifacts / "candidate-nibid"
        self.successor = self.artifacts / "successor-nibid"
        self.rpc = f"http://127.0.0.1:{args.port_base + 7}"
        self.chain_id = f"nibiru-upgrade-e2e-{self.root.name}"
        self.process: subprocess.Popen | None = None
        self.log = None
        self.counter = 0

    def save(self, name: str, value) -> None:
        (self.evidence / f"{name}.json").write_text(json.dumps(value, indent=2) + "\n")

    def command(self, command: list[str], timeout: float = 45) -> str:
        with (self.evidence / "commands.jsonl").open("a") as stream:
            stream.write(json.dumps(command) + "\n")
        result = subprocess.run(command, text=True, capture_output=True, timeout=timeout)
        if result.returncode:
            raise RunError(f"Command failed ({result.returncode}): {command}\n{result.stderr[:1500]}\n{result.stderr[-2000:]}")
        return result.stdout

    def cli(self, binary: Path, *args: str) -> str:
        return self.command([str(binary), *args, "--home", str(self.home)])

    def query(self, binary: Path, *args: str):
        return json.loads(self.cli(binary, "query", *args, "--node", self.rpc, "--output", "json"))

    def rpc_query(self, method: str, **params):
        url = f"{self.rpc}/{method}?{urllib.parse.urlencode(params)}"
        with urllib.request.urlopen(url, timeout=4) as response:
            result = json.load(response)
        if "error" in result:
            raise RunError(f"RPC {method}: {result['error']}")
        return result["result"]

    def height(self) -> int:
        status = self.rpc_query("status")
        return int(status.get("sync_info", status.get("SyncInfo", {}))["latest_block_height"])

    def protocol_version(self) -> int:
        # SetProtocolVersion updates BaseApp's ABCI Info field, not the
        # independently stored consensus-parameter version in block headers.
        return int(self.rpc_query("abci_info")["response"].get("app_version", 0))

    def contract_snapshot(self, binary: Path, label: str):
        address = self.query(binary, "upgrade", "status")["contract_address"]
        response = self.query(binary, "wasm", "contract-state", "smart", address, '{"snapshot":{}}')
        snapshot = response.get("data", response)
        self.save(label, snapshot)
        return snapshot

    def wait(self, description: str, check, timeout: float | None = None):
        deadline = time.monotonic() + (timeout or self.args.timeout)
        last = ""
        while time.monotonic() < deadline:
            if self.process and self.process.poll() is not None:
                raise RunError(f"Cosmovisor exited while waiting for {description}; see node.log")
            try:
                value = check()
                if value:
                    return value
            except (RunError, OSError, ValueError, KeyError) as error:
                last = str(error)
            time.sleep(0.5)
        raise RunError(f"Timed out waiting for {description}: {last}")

    def prepare(self) -> None:
        arch = {"x86_64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(platform.machine())
        system = platform.system().lower()
        if arch is None or system not in ("linux", "darwin"):
            raise RunError("Use a supported Linux/macOS amd64/arm64 host")
        cache = self.args.baseline_cache or find_cache()
        archive = cache / f"nibid_{BASELINE_VERSION}_{system}_{arch}.tar.gz"
        manifest = cache / f"nibid_{BASELINE_VERSION}_checksums.txt"
        extract_verified_binary(archive, manifest, self.baseline)
        baseline_version = self.command([str(self.baseline), "version"]).strip()
        if baseline_version.removeprefix("v") != BASELINE_VERSION:
            raise RunError(f"Unexpected baseline version {baseline_version}")
        versions = {"baseline": {"version": baseline_version, "sha256": file_hash(self.baseline)}}
        for label, path, destination in (
            ("candidate", self.args.candidate, self.candidate),
            ("successor", self.args.successor, self.successor),
        ):
            if path:
                shutil.copy2(path, destination)
                destination.chmod(0o755)
                versions[label] = {
                    "version": self.command([str(destination), "version"]).strip(),
                    "sha256": file_hash(destination),
                }
        self.save("versions", versions)
        self.cli(self.baseline, "init", self.chain_id, "--chain-id", self.chain_id, "--default-denom", "unibi")
        # Key output includes a mnemonic. Keep it out of evidence and terminal output.
        self.cli(self.baseline, "keys", "add", "validator", "--keyring-backend", "test", "--output", "json")
        validator = self.cli(self.baseline, "keys", "show", "validator", "-a", "--keyring-backend", "test").strip()
        self.cli(self.baseline, "genesis", "add-genesis-account", validator, "10000000000000unibi")
        genesis_path = self.home / "config/genesis.json"
        genesis = json.loads(genesis_path.read_text())
        gov = genesis["app_state"]["gov"]["params"]
        gov.update(voting_period="10s", max_deposit_period="5s", min_deposit=[{"denom": "unibi", "amount": "1000"}])
        genesis["app_state"]["sudo"]["sudoers"]["root"] = validator
        genesis_path.write_text(json.dumps(genesis, indent=2) + "\n")
        self.cli(self.baseline, "genesis", "gentx", "validator", "1000000000000unibi", "--chain-id", self.chain_id, "--keyring-backend", "test")
        self.cli(self.baseline, "genesis", "collect-gentxs")
        config = self.home / "config/config.toml"
        config.write_text(set_toml_value(config.read_text(), "consensus", "timeout_commit", '"500ms"'))
        self.cli(self.baseline, "genesis", "validate-genesis")
        self.save("run", {"chain_id": self.chain_id, "home": str(self.home), "rpc": self.rpc, "validator": validator,
                          "port_base": self.args.port_base, "default_delay_blocks": 1000})

    def start(self, genesis_binary: Path | None = None) -> None:
        require_unused_ports([self.args.port_base + n for n in (6, 7, 8)])
        if self.args.mode == "run" and (self.args.candidate is None or self.args.successor is None):
            raise RunError("run requires --candidate and --successor")
        cosmo = shutil.which(self.args.cosmovisor)
        if cosmo is None:
            raise RunError("Pinned Cosmovisor v1.7.1 is required")
        for binary, name in ((genesis_binary or self.baseline, "genesis"), (self.candidate, f"upgrades/{self.args.enabling_name}"),
                             (self.successor, f"upgrades/{NEXT_UPGRADE}")):
            if not binary.is_file():
                continue
            directory = self.home / "cosmovisor" / name / "bin"
            directory.mkdir(parents=True, exist_ok=True)
            shutil.copy2(binary, directory / "nibid")
        env = os.environ | {"DAEMON_HOME": str(self.home), "DAEMON_NAME": "nibid",
                            "DAEMON_ALLOW_DOWNLOAD_BINARIES": "false", "DAEMON_RESTART_AFTER_UPGRADE": "true",
                            "UNSAFE_SKIP_BACKUP": "true", "DAEMON_POLL_INTERVAL": "300ms"}
        version = subprocess.run([cosmo, "version"], env=env, text=True, capture_output=True, timeout=15)
        if "v1.7.1" not in version.stdout + version.stderr:
            raise RunError("Expected Cosmovisor v1.7.1")
        self.save("cosmovisor", {"version_output": version.stdout + version.stderr, "path": cosmo})
        self.log = (self.root / "node.log").open("a")
        command = [cosmo, "run", "start", "--home", str(self.home), "--rpc.laddr", f"tcp://127.0.0.1:{self.args.port_base + 7}",
                   "--p2p.laddr", f"tcp://127.0.0.1:{self.args.port_base + 6}", "--grpc.address", f"127.0.0.1:{self.args.port_base + 8}",
                   "--grpc.enable=true", "--grpc-web.enable=false", "--api.enable=false", "--json-rpc.enable=false", "--minimum-gas-prices", "0unibi"]
        self.process = subprocess.Popen(command, env=env, stdout=self.log, stderr=subprocess.STDOUT, start_new_session=True)
        self.wait("first committed baseline block", lambda: self.height() >= 2, timeout=90)

    def smoke(self) -> None:
        self.start()
        self.wait("five baseline blocks", lambda: self.height() >= 5, timeout=90)
        self.save("baseline-status", self.rpc_query("status"))
        self.save("baseline-gov-params", self.query(self.baseline, "gov", "params"))
        authority = self.governance_authority()
        self.proposal(self.baseline, "smoke-schedule", self.plan_message(authority, "e2e-smoke", height=self.height() + 500))
        self.proposal(self.baseline, "smoke-cancel", {"@type": "/cosmos.upgrade.v1beta1.MsgCancelUpgrade", "authority": authority})
        self.expect_no_plan(self.baseline)
        self.save("result", {"passed": True, "mode": "smoke", "height": self.height(), "run_dir": str(self.root)})

    def stop(self) -> None:
        if self.process:
            with contextlib.suppress(ProcessLookupError):
                os.killpg(self.process.pid, signal.SIGTERM)
            try:
                self.process.wait(timeout=15)
            except subprocess.TimeoutExpired:
                with contextlib.suppress(ProcessLookupError):
                    os.killpg(self.process.pid, signal.SIGKILL)
                self.process.wait(timeout=5)
            self.process = None
        if self.log:
            self.log.close()
            self.log = None

    def tx(self, binary: Path, label: str, *args: str):
        response = json.loads(self.cli(binary, "tx", *args, "--from", "validator", "--keyring-backend", "test",
                                      "--chain-id", self.chain_id, "--node", self.rpc, "--gas", "2000000",
                                      "--fees", "50000unibi", "--yes", "--output", "json"))
        response = response.get("tx_response", response)
        if int(response.get("code", 0)):
            raise RunError(f"Broadcast failed: {response}")
        tx_hash = response["txhash"]
        receipt = self.wait(f"committed {label}", lambda: self.query(binary, "tx", tx_hash), timeout=45)
        receipt = receipt.get("tx_response", receipt)
        self.save(label, receipt)
        if int(receipt.get("code", 0)):
            raise RunError(f"Transaction {label} failed: {receipt.get('raw_log')}")
        return receipt

    def proposal(self, binary: Path, label: str, message: dict) -> int:
        self.counter += 1
        stem = f"{self.counter:02d}-{label}"
        document = {"messages": [message], "metadata": "", "deposit": "1000unibi", "title": label, "summary": label}
        path = self.evidence / f"{stem}-proposal.json"
        path.write_text(json.dumps(document, indent=2) + "\n")
        receipt = self.tx(binary, f"{stem}-submit", "gov", "submit-proposal", str(path))
        ids = [attribute["value"] for event in receipt.get("events", []) for attribute in event.get("attributes", [])
               if attribute.get("key") == "proposal_id"]
        if not ids:
            raise RunError("Committed proposal transaction has no proposal_id event")
        proposal_id = int(ids[0])
        self.tx(binary, f"{stem}-vote", "gov", "vote", str(proposal_id), "yes")

        def finalized():
            proposal = self.query(binary, "gov", "proposal", str(proposal_id))
            proposal = proposal.get("proposal", proposal)
            status = proposal.get("status")
            if status in ("PROPOSAL_STATUS_REJECTED", "PROPOSAL_STATUS_FAILED", 4, 5):
                raise RunError(f"Proposal {proposal_id} failed: {proposal}")
            return proposal if status in ("PROPOSAL_STATUS_PASSED", 3) else None

        final = self.wait(f"proposal {proposal_id} passage", finalized, timeout=75)
        self.save(f"{stem}-passed", final)
        return proposal_id

    def plan_message(self, authority: str, name: str, *, height: int = 0, target: str | None = None) -> dict:
        plan = {"name": name, "height": str(height), "info": json.dumps({"binaries": {"docker": "nibiru-upgrade-e2e:local"}})}
        if target:
            plan["time"] = target
        return {"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", "authority": authority, "plan": plan}

    def capture_blocks(self, label: str, height: int) -> None:
        for h in range(max(1, height - 1), height + 2):
            self.wait(f"block {h}", lambda: self.height() >= h)
            for method in ("block", "block_results"):
                self.save(f"{label}-{method}-{h}", self.rpc_query(method, height=h))

    def governance_authority(self) -> str:
        gov = self.query(self.baseline, "auth", "module-account", "gov")
        account = gov.get("account", gov)
        authority = account.get("base_account", account).get("address")
        if not authority:
            raise RunError(f"Could not find governance module address: {gov}")
        return authority

    def expect_no_plan(self, binary: Path) -> None:
        try:
            plan = self.query(binary, "upgrade", "plan")
        except RunError as error:
            if "no upgrade scheduled" in str(error):
                return
            raise
        raise RunError(f"Expected no concrete execution plan, got {plan}")

    def export_and_import(self, history, versions, expected_protocol: int) -> None:
        export_path = self.evidence / "exported-genesis.json"
        self.command([str(self.successor), "export", "--home", str(self.home), "--output-document", str(export_path)], timeout=120)
        exported = json.loads(export_path.read_text())
        original_home = self.home
        self.home = self.root / "imported-node"
        self.cli(self.successor, "init", "imported-upgrade-state", "--chain-id", self.chain_id, "--default-denom", "unibi")
        shutil.copy2(export_path, self.home / "config/genesis.json")
        for filename in ("config.toml", "app.toml", "priv_validator_key.json"):
            shutil.copy2(original_home / "config" / filename, self.home / "config" / filename)
        # Original process is stopped. Preserve signer height and original chain ID.
        shutil.copy2(original_home / "data/priv_validator_state.json", self.home / "data/priv_validator_state.json")
        self.start(genesis_binary=self.successor)
        first_height = int(exported["initial_height"])
        self.wait("three blocks from imported genesis", lambda: self.height() >= first_height + 2)
        if self.query(self.successor, "upgrade", "history") != history:
            raise RunError("Genesis import changed upgrade history")
        if self.query(self.successor, "upgrade", "module_versions") != versions:
            raise RunError("Genesis import changed module versions")
        if self.protocol_version() != expected_protocol:
            raise RunError("Genesis import did not restore the native protocol version")
        self.save("imported-status", self.query(self.successor, "upgrade", "status"))
        self.save("imported-rpc-status", self.rpc_query("status"))
        self.save("imported-abci-info", self.rpc_query("abci_info"))
        self.stop()
        restored_path = self.evidence / "reexported-genesis.json"
        self.command([str(self.successor), "export", "--home", str(self.home), "--output-document", str(restored_path)], timeout=120)
        restored = json.loads(restored_path.read_text())
        # Compare canonical contract bytes, excluding host-generated metadata.
        def records(genesis):
            return {contract["contract_address"]: contract["contract_state"]
                    for contract in genesis["app_state"]["wasm"]["contracts"]}
        if records(exported) != records(restored):
            raise RunError("Wasm contract records changed across genesis export/import")
        self.save("genesis-roundtrip", {"passed": True, "initial_height": first_height,
                                        "contract_count": len(records(exported))})

    def run(self) -> None:
        self.start()
        authority = self.governance_authority()
        baseline_protocol = self.protocol_version()
        self.save("baseline-module-versions", self.query(self.baseline, "upgrade", "module_versions"))
        height = self.height() + 100
        self.proposal(self.baseline, "enable-x-upgrade", self.plan_message(authority, self.args.enabling_name, height=height))
        self.wait("enabling upgrade and five later blocks", lambda: self.height() >= height + 5)
        self.capture_blocks("enable", height)
        current = (self.home / "cosmovisor/current").resolve()
        if current.name != self.args.enabling_name:
            raise RunError(f"Cosmovisor did not switch to enabling binary: {current}")
        self.save("enabled-status", self.query(self.candidate, "upgrade", "status"))
        self.save("enabled-history", self.query(self.candidate, "upgrade", "history"))
        self.save("enabled-module-versions", self.query(self.candidate, "upgrade", "module_versions"))
        if self.protocol_version() != baseline_protocol + 1:
            raise RunError("Enabling upgrade did not increment protocol version exactly once")
        snapshot = self.contract_snapshot(self.candidate, "enabled-contract-snapshot")
        if int(snapshot.get("protocol_version") or 0) != baseline_protocol + 1:
            raise RunError("Canonical contract protocol version does not match the native host field")
        applied = self.query(self.candidate, "upgrade", "applied", self.args.enabling_name)
        if int(applied["header"]["height"]) != height:
            raise RunError("Enabling upgrade completed at an unexpected height")
        self.save("enabling-applied", applied)
        shutil.copy2(self.home / "data/upgrade-info.json", self.evidence / "enabling-upgrade-info.json")

        # Exercise legacy cancellation through native governance before time selection.
        self.proposal(self.candidate, "schedule-cancelled", self.plan_message(authority, "e2e-cancelled", height=self.height() + 500))
        self.proposal(self.candidate, "cancel-upgrade", {"@type": "/cosmos.upgrade.v1beta1.MsgCancelUpgrade", "authority": authority})
        self.expect_no_plan(self.candidate)
        self.save("after-cancel-status", self.query(self.candidate, "upgrade", "status"))
        self.proposal(self.candidate, "schedule-replaced", self.plan_message(authority, "e2e-replaced", height=self.height() + 500))

        block = self.rpc_query("block")["block"]["header"]
        block_time = dt.datetime.fromisoformat(block["time"].replace("Z", "+00:00"))
        target = (block_time + dt.timedelta(seconds=30)).isoformat(timespec="seconds").replace("+00:00", "Z")
        self.proposal(self.candidate, "time-successor", self.plan_message(authority, NEXT_UPGRADE, target=target))
        self.save("pending-time-status", self.query(self.candidate, "upgrade", "status"))

        def concrete_plan():
            try:
                plan = self.query(self.candidate, "upgrade", "plan")
            except RunError:
                return None
            plan = plan.get("plan", plan)
            return plan if plan.get("name") == NEXT_UPGRADE and int(plan.get("height", 0)) > 0 else None

        plan = self.wait("time target selecting an execution height", concrete_plan, timeout=90)
        selected_height = int(plan["height"])
        self.save("selected-plan", plan)
        self.save("selection-observation", {"observed_height": self.height(), "execution_height": selected_height, "target_time": target})
        selection_height = selected_height - 1000
        if selection_height <= 0:
            raise RunError("Selected height cannot contain the default 1,000-block delay")
        self.capture_blocks("selection", selection_height)
        previous_header = self.rpc_query("block", height=selection_height - 1)["block"]["header"]
        selected_header = self.rpc_query("block", height=selection_height)["block"]["header"]
        as_time = lambda value: dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
        if not as_time(previous_header["time"]) < as_time(target) <= as_time(selected_header["time"]):
            raise RunError("Concrete execution height does not equal first target-reaching height plus 1,000")
        self.wait("successor upgrade and five later blocks", lambda: self.height() >= selected_height + 5)
        self.capture_blocks("successor", selected_height)
        if (self.home / "cosmovisor/current").resolve().name != NEXT_UPGRADE:
            raise RunError("Cosmovisor did not switch to successor binary")
        self.save("final-status", self.query(self.successor, "upgrade", "status"))
        self.save("final-history", self.query(self.successor, "upgrade", "history"))
        self.save("final-module-versions", self.query(self.successor, "upgrade", "module_versions"))
        if self.protocol_version() != baseline_protocol + 2:
            raise RunError("Successor upgrade did not increment protocol version exactly once")
        snapshot = self.contract_snapshot(self.successor, "final-contract-snapshot")
        if int(snapshot.get("protocol_version") or 0) != baseline_protocol + 2:
            raise RunError("Successor contract protocol version does not match the native host field")
        self.expect_no_plan(self.successor)
        shutil.copy2(self.home / "data/upgrade-info.json", self.evidence / "successor-upgrade-info.json")
        history = self.query(self.successor, "upgrade", "history")
        versions = self.query(self.successor, "upgrade", "module_versions")
        restart_height = self.height()
        self.stop()
        self.start()
        self.wait("continued blocks after restart", lambda: self.height() >= restart_height + 3)
        if self.query(self.successor, "upgrade", "history") != history:
            raise RunError("Upgrade history changed after process restart")
        if self.query(self.successor, "upgrade", "module_versions") != versions:
            raise RunError("Module versions changed after process restart")
        if self.protocol_version() != baseline_protocol + 2:
            raise RunError("Protocol version was not restored after process restart")
        self.save("restart-status", self.rpc_query("status"))
        self.stop()
        # Export is intentionally nonzero-height so height-based history is not rebased.
        self.export_and_import(history, versions, baseline_protocol + 2)
        self.save("result", {"passed": True, "enabling_height": height, "successor_height": selected_height,
                             "run_dir": str(self.root), "note": "Local Cosmovisor, restart, and genesis round-trip acceptance; Cosmopilot and injected failure/replay are separate checks."})


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("prepare", "smoke", "run"))
    parser.add_argument("--candidate", type=Path, help="Enabling binary, without the successor fixture handler")
    parser.add_argument("--successor", type=Path, help="Candidate built with nibiru_upgrade_e2e_next")
    parser.add_argument("--baseline-cache", type=Path, help="Verified GitHub v2.21.2 cache directory")
    parser.add_argument("--run-dir", type=Path, help="New directory to create; an existing path is rejected")
    parser.add_argument("--port-base", type=int, default=27650)
    parser.add_argument("--timeout", type=int, default=1800, help="Height wait deadline in seconds")
    parser.add_argument("--cosmovisor", default="cosmovisor")
    parser.add_argument("--enabling-name", default="v2.22.0")
    args = parser.parse_args()
    if not 1024 <= args.port_base <= 65527 or args.timeout <= 0:
        parser.error("port-base must be 1024..65527 and timeout must be positive")
    for label in ("candidate", "successor"):
        path = getattr(args, label)
        if path:
            setattr(args, label, path.expanduser().resolve(strict=True))
    runner = Runner(args)
    print(f"Upgrade experiment: {runner.root}", flush=True)
    try:
        runner.prepare()
        if args.mode == "run":
            runner.run()
        elif args.mode == "smoke":
            runner.smoke()
        print(f"Completed {args.mode}; evidence retained at {runner.evidence}", flush=True)
        return 0
    except (RunError, OSError, subprocess.SubprocessError, ValueError) as error:
        runner.save("result", {"passed": False, "error": str(error), "run_dir": str(runner.root)})
        print(f"ERROR: {error}\nEvidence retained at {runner.root}", file=sys.stderr)
        return 1
    finally:
        runner.stop()


if __name__ == "__main__":
    raise SystemExit(main())
