#!/usr/bin/env bun

import { join } from "path"

type TxEvent = {
  type: string
  attributes: { key: string; value: string }[]
}

export type TxResRaw = {
  height: string
  txhash: string
  codespace: string
  code: number
  raw_log: string
  events: TxEvent[]
  gas_wanted: string
  gas_used: string
}

const CONFIG = {
  node: "http://localhost:26657",
  chainId: "nibiru-localnet-0",
  fees: "750000unibi",
  gas: "30000000",
  keyringBackend: "test",
  sudoFromKey: "validator",
}

const TX_FLAGS = `--fees ${CONFIG.fees} --gas ${CONFIG.gas} --yes --keyring-backend ${CONFIG.keyringBackend} --chain-id ${CONFIG.chainId} --node ${CONFIG.node}`

export async function execCommand(
  command: string,
): Promise<{ stdout: string; stderr: string }> {
  const proc = Bun.spawn(["bash", "-c", command], {
    env: process.env as Record<string, string>,
    stdout: "pipe",
    stderr: "pipe",
  })

  const stdout = await new Response(proc.stdout).text()
  const stderr = await new Response(proc.stderr).text()

  const exitCode = await proc.exited
  if (exitCode !== 0) {
    throw new Error(
      `Command exited with code ${exitCode}: ${stderr || stdout || command}`,
    )
  }

  return { stdout, stderr }
}

export async function waitForTx(txhash: string): Promise<TxResRaw> {
  // Allow time for the tx to be included in a block.
  await new Promise((resolve) => setTimeout(resolve, 6000))

  const { stdout } = await execCommand(
    `nibid q tx ${txhash} --node ${CONFIG.node}`,
  )

  const jsonResult = JSON.parse(stdout) as TxResRaw
  return jsonResult
}

export function ensureTxOk(result: TxResRaw): void {
  const rawLog = result.raw_log || ""

  if (rawLog.includes("failed to execute message")) {
    throw new Error(`Transaction failed: ${rawLog}`)
  }

  if (result.code !== 0) {
    throw new Error(
      `Transaction failed with code ${result.code}: ${rawLog || result.txhash}`,
    )
  }
}

export async function addZeroGasContract(
  evmAddr: string,
): Promise<void> {
  const actors = JSON.stringify({
    senders: [] as string[],
    contracts: [] as string[],
    always_zero_gas_contracts: [evmAddr],
  })

  const cmd = `nibid tx sudo edit-zero-gas '${actors}' --from ${CONFIG.sudoFromKey} ${TX_FLAGS}`

  const { stdout } = await execCommand(cmd)
  const txhash = JSON.parse(stdout).txhash as string

  const txRes = await waitForTx(txhash)
  ensureTxOk(txRes)
}

