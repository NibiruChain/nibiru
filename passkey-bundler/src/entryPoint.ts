import { Interface } from "ethers"

const USER_OP_TUPLE =
  "(address sender,uint256 nonce,bytes initCode,bytes callData,uint256 callGasLimit,uint256 verificationGasLimit,uint256 preVerificationGas,uint256 maxFeePerGas,uint256 maxPriorityFeePerGas,bytes paymasterAndData,bytes signature)"

export const ENTRY_POINT_V06_ABI = [
  `function handleOps(${USER_OP_TUPLE}[] ops, address payable beneficiary)`,
  "function balanceOf(address account) view returns (uint256)",
  `function simulateValidation(${USER_OP_TUPLE} userOp)`,
  `function simulateHandleOp(${USER_OP_TUPLE} op, address target, bytes targetCallData)`,
  "event UserOperationEvent(bytes32 indexed userOpHash, address indexed sender, address indexed paymaster, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed)",
  "event UserOperationRevertReason(bytes32 indexed userOpHash, address indexed sender, uint256 nonce, bytes revertReason)",
  "error FailedOp(uint256 opIndex, string reason)",
  "error SignatureValidationFailed(address aggregator)",
  "error ValidationResult((uint256 preOpGas,uint256 prefund,bool sigFailed,uint48 validAfter,uint48 validUntil,bytes paymasterContext) returnInfo,(uint256 stake,uint256 unstakeDelaySec) senderInfo,(uint256 stake,uint256 unstakeDelaySec) factoryInfo,(uint256 stake,uint256 unstakeDelaySec) paymasterInfo)",
  "error ValidationResultWithAggregation((uint256 preOpGas,uint256 prefund,bool sigFailed,uint48 validAfter,uint48 validUntil,bytes paymasterContext) returnInfo,(uint256 stake,uint256 unstakeDelaySec) senderInfo,(uint256 stake,uint256 unstakeDelaySec) factoryInfo,(uint256 stake,uint256 unstakeDelaySec) paymasterInfo,(address aggregator,(uint256 stake,uint256 unstakeDelaySec) stakeInfo) aggregatorInfo)",
  "error ExecutionResult(uint256 preOpGas, uint256 paid, uint48 validAfter, uint48 validUntil, bool targetSuccess, bytes targetResult)",
] as const

export const entryPointInterfaceV06 = new Interface(ENTRY_POINT_V06_ABI)

export function extractRevertData(err: unknown): string | undefined {
  const e = err as any
  return (
    e?.data ??
    e?.error?.data ??
    e?.info?.error?.data ??
    e?.info?.error?.data?.data ??
    e?.info?.error?.data?.originalError?.data ??
    undefined
  )
}
