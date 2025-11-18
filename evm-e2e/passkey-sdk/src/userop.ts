import { AbiCoder, BytesLike, keccak256, toUtf8Bytes, zeroPadValue } from "ethers"

export interface UserOperation {
  sender: string
  nonce: bigint
  initCode: string
  callData: string
  callGasLimit: bigint
  verificationGasLimit: bigint
  preVerificationGas: bigint
  maxFeePerGas: bigint
  maxPriorityFeePerGas: bigint
  paymasterAndData: string
  signature: string
}

export function encodeSignature(rs: { r: BytesLike; s: BytesLike }): string {
  const abi = new AbiCoder()
  return abi.encode(["bytes32", "bytes32"], [rs.r, rs.s])
}

export function packUserOp(op: UserOperation): string {
  const abi = new AbiCoder()
  return abi.encode(
    [
      "address",
      "uint256",
      "bytes",
      "bytes",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
      "bytes",
      "bytes",
    ],
    [
      op.sender,
      op.nonce,
      op.initCode,
      op.callData,
      op.callGasLimit,
      op.verificationGasLimit,
      op.preVerificationGas,
      op.maxFeePerGas,
      op.maxPriorityFeePerGas,
      op.paymasterAndData,
      op.signature,
    ],
  )
}

// Compute the userOpHash per ERC-4337 v0.6 spec.
export function getUserOpHash(op: UserOperation, entryPoint: string, chainId: bigint): string {
  const userOpPack = packUserOp(op)
  const userOpHash = keccak256(userOpPack)
  const enc = new AbiCoder().encode(["bytes32", "address", "uint256"], [userOpHash, entryPoint, chainId])
  return keccak256(enc)
}

export function defaultUserOp(from: string): UserOperation {
  return {
    sender: from,
    nonce: 0n,
    initCode: "0x",
    callData: "0x",
    callGasLimit: 100000n,
    verificationGasLimit: 100000n,
    preVerificationGas: 21000n,
    maxFeePerGas: 1_000_000_000_000n,
    maxPriorityFeePerGas: 0n,
    paymasterAndData: "0x",
    signature: "0x",
  }
}

export function stringToBytes32(str: string): string {
  return zeroPadValue(toUtf8Bytes(str), 32)
}
