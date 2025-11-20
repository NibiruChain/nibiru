import { AbiCoder, BytesLike, keccak256, toBeHex, toUtf8Bytes, zeroPadValue } from "ethers"

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

export interface RpcUserOperation {
  sender: string
  nonce: string
  initCode: string
  callData: string
  callGasLimit: string
  verificationGasLimit: string
  preVerificationGas: string
  maxFeePerGas: string
  maxPriorityFeePerGas: string
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
      "bytes32",
      "bytes32",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
      "bytes32",
    ],
    [
      op.sender,
      op.nonce,
      keccak256(op.initCode),
      keccak256(op.callData),
      op.callGasLimit,
      op.verificationGasLimit,
      op.preVerificationGas,
      op.maxFeePerGas,
      op.maxPriorityFeePerGas,
      keccak256(op.paymasterAndData),
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
    callGasLimit: 200000n,
    verificationGasLimit: 300000n,
    preVerificationGas: 50000n,
    maxFeePerGas: 2_000_000_000n,
    maxPriorityFeePerGas: 1_000_000_000n,
    paymasterAndData: "0x",
    signature: "0x",
  }
}

export function stringToBytes32(str: string): string {
  return zeroPadValue(toUtf8Bytes(str), 32)
}

export function toRpcUserOperation(op: UserOperation): RpcUserOperation {
  return {
    sender: op.sender,
    nonce: toBeHex(op.nonce),
    initCode: op.initCode,
    callData: op.callData,
    callGasLimit: toBeHex(op.callGasLimit),
    verificationGasLimit: toBeHex(op.verificationGasLimit),
    preVerificationGas: toBeHex(op.preVerificationGas),
    maxFeePerGas: toBeHex(op.maxFeePerGas),
    maxPriorityFeePerGas: toBeHex(op.maxPriorityFeePerGas),
    paymasterAndData: op.paymasterAndData,
    signature: op.signature,
  }
}

export function rpcUserOpToStruct(rpc: RpcUserOperation): UserOperation {
  return {
    sender: rpc.sender,
    nonce: BigInt(rpc.nonce),
    initCode: rpc.initCode,
    callData: rpc.callData,
    callGasLimit: BigInt(rpc.callGasLimit),
    verificationGasLimit: BigInt(rpc.verificationGasLimit),
    preVerificationGas: BigInt(rpc.preVerificationGas),
    maxFeePerGas: BigInt(rpc.maxFeePerGas),
    maxPriorityFeePerGas: BigInt(rpc.maxPriorityFeePerGas),
    paymasterAndData: rpc.paymasterAndData,
    signature: rpc.signature,
  }
}
