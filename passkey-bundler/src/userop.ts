import { AbiCoder, keccak256, toBeHex, zeroPadValue } from "ethers"
import { z } from "zod"

import { RpcUserOperation, UserOperation } from "./types"

const hexString = z.string().regex(/^0x[0-9a-fA-F]*$/)
const address = z.string().regex(/^0x[a-fA-F0-9]{40}$/)

const rpcUserOpSchema = z.object({
  sender: address,
  nonce: hexString,
  initCode: hexString,
  callData: hexString,
  callGasLimit: hexString,
  verificationGasLimit: hexString,
  preVerificationGas: hexString,
  maxFeePerGas: hexString,
  maxPriorityFeePerGas: hexString,
  paymasterAndData: hexString,
  signature: hexString,
})

export function parseRpcUserOp(input: unknown): { rpcUserOp: RpcUserOperation; userOp: UserOperation } {
  const rpcUserOp = rpcUserOpSchema.parse(input)
  const userOp = rpcUserOpToStruct(rpcUserOp)
  if (userOp.maxPriorityFeePerGas > userOp.maxFeePerGas) {
    throw new Error("maxPriorityFeePerGas cannot exceed maxFeePerGas")
  }
  return { rpcUserOp, userOp }
}

export function rpcUserOpToStruct(rpc: RpcUserOperation): UserOperation {
  return {
    sender: rpc.sender,
    nonce: hexToBigInt(rpc.nonce),
    initCode: normalizeHex(rpc.initCode),
    callData: normalizeHex(rpc.callData),
    callGasLimit: hexToBigInt(rpc.callGasLimit),
    verificationGasLimit: hexToBigInt(rpc.verificationGasLimit),
    preVerificationGas: hexToBigInt(rpc.preVerificationGas),
    maxFeePerGas: hexToBigInt(rpc.maxFeePerGas),
    maxPriorityFeePerGas: hexToBigInt(rpc.maxPriorityFeePerGas),
    paymasterAndData: normalizeHex(rpc.paymasterAndData),
    signature: normalizeHex(rpc.signature),
  }
}

export function toRpcUserOp(userOp: UserOperation): RpcUserOperation {
  return {
    sender: userOp.sender,
    nonce: toHex(userOp.nonce),
    initCode: userOp.initCode,
    callData: userOp.callData,
    callGasLimit: toHex(userOp.callGasLimit),
    verificationGasLimit: toHex(userOp.verificationGasLimit),
    preVerificationGas: toHex(userOp.preVerificationGas),
    maxFeePerGas: toHex(userOp.maxFeePerGas),
    maxPriorityFeePerGas: toHex(userOp.maxPriorityFeePerGas),
    paymasterAndData: userOp.paymasterAndData,
    signature: userOp.signature,
  }
}

export function getUserOpHash(userOp: UserOperation, entryPoint: string, chainId: bigint): string {
  const abi = new AbiCoder()
  const userOpPack = abi.encode(
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
      userOp.sender,
      userOp.nonce,
      keccak256(userOp.initCode),
      keccak256(userOp.callData),
      userOp.callGasLimit,
      userOp.verificationGasLimit,
      userOp.preVerificationGas,
      userOp.maxFeePerGas,
      userOp.maxPriorityFeePerGas,
      keccak256(userOp.paymasterAndData),
    ],
  )
  const userOpHash = keccak256(userOpPack)
  const enc = abi.encode(["bytes32", "address", "uint256"], [userOpHash, entryPoint, chainId])
  return keccak256(enc)
}

export function toHex(value: bigint): string {
  return toBeHex(value)
}

export function stringToBytes32(str: string): string {
  const encoder = new TextEncoder()
  return normalizeHex(zeroPadValue(encoder.encode(str), 32))
}

function hexToBigInt(value: string): bigint {
  if (!value.startsWith("0x")) {
    throw new Error(`expected hex string, got ${value}`)
  }
  if (value === "0x") {
    return 0n
  }
  return BigInt(value)
}

function normalizeHex(value: string): string {
  return value === "0x" ? "0x" : value.toLowerCase()
}
