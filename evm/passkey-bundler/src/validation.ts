import { Contract, Interface, JsonRpcProvider } from "ethers"

import { entryPointInterfaceV06, extractRevertData } from "./entryPoint"
import { UserOperation } from "./types"

const standardErrorInterface = new Interface(["error Error(string)", "error Panic(uint256)"])

export interface SimulateValidationResult {
  returnInfo: {
    preOpGas: bigint
    prefund: bigint
    sigFailed: boolean
    validAfter: bigint
    validUntil: bigint
    paymasterContext: string
  }
  aggregator?: string
}

export async function simulateValidationOrThrow(opts: {
  provider: JsonRpcProvider
  entryPoint: string
  userOp: UserOperation
}): Promise<SimulateValidationResult> {
  const contract = new Contract(opts.entryPoint, entryPointInterfaceV06, opts.provider)
  try {
    await contract.simulateValidation.staticCall(opts.userOp)
  } catch (err) {
    const data = extractRevertData(err)
    if (!data || data === "0x") {
      throw new Error((err as Error).message ?? "simulateValidation failed")
    }

    const parsed = tryParseError(data)
    if (!parsed) {
      throw new Error((err as Error).message ?? "simulateValidation failed")
    }

    switch (parsed.name) {
      case "ValidationResult": {
        const returnInfo = parsed.args.returnInfo as any
        if (returnInfo.sigFailed) {
          throw new Error("Signature validation failed")
        }
        return { returnInfo }
      }
      case "ValidationResultWithAggregation": {
        const aggregatorInfo = parsed.args.aggregatorInfo as any
        const aggregator = (aggregatorInfo?.aggregator as string | undefined) ?? "0x"
        throw new Error(`Signature aggregator not supported: ${aggregator}`)
      }
      case "FailedOp": {
        const reason = (parsed.args.reason as string | undefined) ?? "validation failed"
        throw new Error(reason)
      }
      case "SignatureValidationFailed":
        throw new Error("Signature validation failed")
      case "Error":
        throw new Error((parsed.args[0] as string | undefined) ?? "Execution reverted")
      case "Panic":
        throw new Error(`Panic(${String(parsed.args[0])})`)
      default:
        throw new Error(`${parsed.name} revert`)
    }
  }

  throw new Error("simulateValidation unexpectedly returned without reverting")
}

export async function estimateUserOperationGas(opts: {
  provider: JsonRpcProvider
  entryPoint: string
  beneficiary: string
  userOp: UserOperation
}): Promise<{ callGasLimit: bigint; verificationGasLimit: bigint; preVerificationGas: bigint }> {
  const callGasLimit = await estimateCallGasLimit({
    provider: opts.provider,
    entryPoint: opts.entryPoint,
    sender: opts.userOp.sender,
    callData: opts.userOp.callData,
    fallback: opts.userOp.callGasLimit > 0n ? opts.userOp.callGasLimit : 500_000n,
  })

  const preVerificationGas =
    opts.userOp.preVerificationGas > 0n
      ? opts.userOp.preVerificationGas
      : estimatePreVerificationGas({
          beneficiary: opts.beneficiary,
          userOp: { ...opts.userOp, callGasLimit, verificationGasLimit: 0n, preVerificationGas: 0n },
        })

  if (opts.userOp.verificationGasLimit > 0n) {
    return { callGasLimit, verificationGasLimit: opts.userOp.verificationGasLimit, preVerificationGas }
  }

  const baseline = 500_000n
  const candidate: UserOperation = {
    ...opts.userOp,
    callGasLimit,
    verificationGasLimit: baseline,
    preVerificationGas,
  }

  try {
    const sim = await simulateValidationOrThrow({
      provider: opts.provider,
      entryPoint: opts.entryPoint,
      userOp: candidate,
    })
    const preOpGas = BigInt(sim.returnInfo.preOpGas)
    const validationGas = preOpGas > preVerificationGas ? preOpGas - preVerificationGas : 0n
    const buffered = addBuffer(validationGas, 20, 50_000n)
    return { callGasLimit, verificationGasLimit: buffered > baseline ? buffered : baseline, preVerificationGas }
  } catch {
    return { callGasLimit, verificationGasLimit: baseline, preVerificationGas }
  }
}

export function estimatePreVerificationGas(opts: { beneficiary: string; userOp: UserOperation }): bigint {
  const data = entryPointInterfaceV06.encodeFunctionData("handleOps", [[opts.userOp], opts.beneficiary])
  const calldataCost = calldataGasCost(data)
  return calldataCost + 30_000n
}

export function calldataGasCost(data: string): bigint {
  if (!data.startsWith("0x")) return 0n
  let gas = 0n
  for (let i = 2; i < data.length; i += 2) {
    const byte = Number.parseInt(data.slice(i, i + 2), 16)
    gas += byte === 0 ? 4n : 16n
  }
  return gas
}

export function addBuffer(value: bigint, percent: number, absolute: bigint): bigint {
  const pct = (value * BigInt(percent)) / 100n
  return value + (pct > absolute ? pct : absolute)
}

async function estimateCallGasLimit(opts: {
  provider: JsonRpcProvider
  entryPoint: string
  sender: string
  callData: string
  fallback: bigint
}): Promise<bigint> {
  if (opts.callData === "0x") return 0n
  try {
    const estimated = await opts.provider.estimateGas({
      from: opts.entryPoint,
      to: opts.sender,
      data: opts.callData,
    })
    return addBuffer(BigInt(estimated), 20, 50_000n)
  } catch {
    return opts.fallback
  }
}

function tryParseError(data: string): { name: string; args: any } | null {
  try {
    const parsed = entryPointInterfaceV06.parseError(data)
    return { name: parsed.name, args: parsed.args as any }
  } catch {
    try {
      const parsed = standardErrorInterface.parseError(data)
      return { name: parsed.name, args: parsed.args as any }
    } catch {
      return null
    }
  }
}
