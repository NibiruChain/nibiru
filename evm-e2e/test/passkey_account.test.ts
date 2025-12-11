import { describe, it } from "@jest/globals"
import type { ChildProcessWithoutNullStreams, SpawnOptions } from "child_process"
import { spawn } from "child_process"
import path from "path"
import { parseEther } from "ethers"

import { account, provider } from "./setup"
import { EntryPointV06__factory, PasskeyAccountFactory__factory } from "../types"

const PASSKEY_SDK_DIR = path.resolve(__dirname, "..", "passkey-sdk")
const NPM_BIN = process.platform === "win32" ? "npm.cmd" : "npm"
const NODE_BIN = "node"
const JSON_RPC_ENDPOINT = process.env.JSON_RPC_ENDPOINT ?? "http://127.0.0.1:8545"
const MNEMONIC = process.env.MNEMONIC
const BUNDLER_DEV_ADDRESS = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
const BUNDLER_PORT = 14437
const PASSKEY_SEED = "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
const PASSKEY_TEST_TIMEOUT = Number(process.env.PASSKEY_TEST_TIMEOUT ?? 240000)

if (!MNEMONIC) {
  throw new Error("MNEMONIC must be set for passkey e2e test")
}

describe(
  "passkey ERC-4337 flow",
  () => {
    it(
      "executes a passkey user operation via local bundler",
      async () => {
        await buildPasskeySdk()
        const { entryPointAddr, factoryAddr } = await deployPasskeyContracts()
        const chainId = BigInt((await provider.getNetwork()).chainId)
        await fundBundlerSigner()

        const bundler = startBundler(entryPointAddr, chainId)
        try {
          await waitForBundlerReady(bundler)
          await runPasskeyScript({ entryPointAddr, factoryAddr })
        } finally {
          await stopProcess(bundler)
        }
      },
      PASSKEY_TEST_TIMEOUT,
    )
  },
)

async function buildPasskeySdk() {
  await runCommand(NPM_BIN, ["run", "build"], {
    cwd: PASSKEY_SDK_DIR,
    env: process.env,
  })
}

async function deployPasskeyContracts() {
  const entryPoint = await new EntryPointV06__factory(account).deploy()
  await entryPoint.waitForDeployment()
  const entryPointAddr = await entryPoint.getAddress()

  const factory = await new PasskeyAccountFactory__factory(account).deploy(entryPointAddr)
  await factory.waitForDeployment()
  const factoryAddr = await factory.getAddress()

  return { entryPointAddr, factoryAddr }
}

async function fundBundlerSigner() {
  const tx = await account.sendTransaction({
    to: BUNDLER_DEV_ADDRESS,
    value: parseEther("10"),
  })
  await tx.wait()
}

function startBundler(entryPointAddr: string, chainId: bigint) {
  const env = {
    ...process.env,
    ENTRY_POINT: entryPointAddr,
    JSON_RPC_ENDPOINT,
    CHAIN_ID: chainId.toString(),
    BUNDLER_PORT: BUNDLER_PORT.toString(),
  }
  const proc = spawn(NODE_BIN, ["dist/local-bundler.js"], {
    cwd: PASSKEY_SDK_DIR,
    env,
    stdio: ["ignore", "pipe", "pipe"],
  })
  pipeOutput(proc, "bundler")
  return proc
}

async function waitForBundlerReady(proc: ChildProcessWithoutNullStreams) {
  await waitForOutput(proc, "Bundler JSON-RPC listening")
}

async function runPasskeyScript(opts: { entryPointAddr: string; factoryAddr: string }) {
  const env = {
    ...process.env,
    JSON_RPC_ENDPOINT,
    BUNDLER_URL: `http://127.0.0.1:${BUNDLER_PORT}`,
    ENTRY_POINT: opts.entryPointAddr,
    FACTORY_ADDR: opts.factoryAddr,
    MNEMONIC,
    PASSKEY_SEED,
    PASSKEY_FUND_VALUE: "2",
  }
  await runCommand(NODE_BIN, ["dist/passkey-e2e.js"], {
    cwd: PASSKEY_SDK_DIR,
    env,
  })
}

async function stopProcess(proc: ChildProcessWithoutNullStreams | null) {
  if (!proc) return
  if (proc.exitCode !== null || proc.signalCode) return
  await new Promise<void>((resolve) => {
    proc.once("exit", () => resolve())
    proc.kill("SIGINT")
    setTimeout(() => {
      if (!proc.killed) proc.kill("SIGKILL")
    }, 2000)
  })
}

function runCommand(cmd: string, args: string[], options: SpawnOptions & { cwd: string }) {
  return new Promise<void>((resolve, reject) => {
    const child = spawn(cmd, args, {
      ...options,
      stdio: "inherit",
    })
    child.on("error", reject)
    child.on("exit", (code) => {
      if (code === 0) resolve()
      else reject(new Error(`${cmd} ${args.join(" ")} exited with code ${code}`))
    })
  })
}

function pipeOutput(proc: ChildProcessWithoutNullStreams, label: string) {
  proc.stdout.on("data", (data) => process.stdout.write(`[${label}] ${data}`))
  proc.stderr.on("data", (data) => process.stderr.write(`[${label}] ${data}`))
}

function waitForOutput(proc: ChildProcessWithoutNullStreams, marker: string, timeoutMs = 20000) {
  return new Promise<void>((resolve, reject) => {
    const onData = (data: Buffer) => {
      if (data.toString().includes(marker)) {
        cleanup()
        resolve()
      }
    }
    const onExit = (code: number | null) => {
      cleanup()
      reject(new Error(`process exited before emitting "${marker}" (code=${code})`))
    }
    const timer = setTimeout(() => {
      cleanup()
      reject(new Error(`timed out waiting for "${marker}"`))
    }, timeoutMs)

    const cleanup = () => {
      clearTimeout(timer)
      proc.stdout.off("data", onData)
      proc.stderr.off("data", onData)
      proc.off("exit", onExit)
    }

    proc.stdout.on("data", onData)
    proc.stderr.on("data", onData)
    proc.once("exit", onExit)
  })
}
