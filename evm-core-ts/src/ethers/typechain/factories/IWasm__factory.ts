/* Autogenerated file. Do not edit manually. */
/* tslint:disable */
/* eslint-disable */

import { Contract, Interface, type ContractRunner } from "ethers"

import type { IWasm, IWasmInterface } from "../IWasm"

const _abi = [
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "string",
        name: "eventType",
        type: "string",
      },
      {
        indexed: false,
        internalType: "string",
        name: "abciEvent",
        type: "string",
      },
    ],
    name: "AbciEvent",
    type: "event",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "contractAddr",
        type: "string",
      },
      {
        internalType: "bytes",
        name: "msgArgs",
        type: "bytes",
      },
      {
        components: [
          {
            internalType: "string",
            name: "denom",
            type: "string",
          },
          {
            internalType: "uint256",
            name: "amount",
            type: "uint256",
          },
        ],
        internalType: "struct INibiruEvm.BankCoin[]",
        name: "funds",
        type: "tuple[]",
      },
    ],
    name: "execute",
    outputs: [
      {
        internalType: "bytes",
        name: "response",
        type: "bytes",
      },
    ],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [
      {
        components: [
          {
            internalType: "string",
            name: "contractAddr",
            type: "string",
          },
          {
            internalType: "bytes",
            name: "msgArgs",
            type: "bytes",
          },
          {
            components: [
              {
                internalType: "string",
                name: "denom",
                type: "string",
              },
              {
                internalType: "uint256",
                name: "amount",
                type: "uint256",
              },
            ],
            internalType: "struct INibiruEvm.BankCoin[]",
            name: "funds",
            type: "tuple[]",
          },
        ],
        internalType: "struct IWasm.WasmExecuteMsg[]",
        name: "executeMsgs",
        type: "tuple[]",
      },
    ],
    name: "executeMulti",
    outputs: [
      {
        internalType: "bytes[]",
        name: "responses",
        type: "bytes[]",
      },
    ],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "admin",
        type: "string",
      },
      {
        internalType: "uint64",
        name: "codeID",
        type: "uint64",
      },
      {
        internalType: "bytes",
        name: "msgArgs",
        type: "bytes",
      },
      {
        internalType: "string",
        name: "label",
        type: "string",
      },
      {
        components: [
          {
            internalType: "string",
            name: "denom",
            type: "string",
          },
          {
            internalType: "uint256",
            name: "amount",
            type: "uint256",
          },
        ],
        internalType: "struct INibiruEvm.BankCoin[]",
        name: "funds",
        type: "tuple[]",
      },
    ],
    name: "instantiate",
    outputs: [
      {
        internalType: "string",
        name: "contractAddr",
        type: "string",
      },
      {
        internalType: "bytes",
        name: "data",
        type: "bytes",
      },
    ],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "contractAddr",
        type: "string",
      },
      {
        internalType: "bytes",
        name: "req",
        type: "bytes",
      },
    ],
    name: "query",
    outputs: [
      {
        internalType: "bytes",
        name: "response",
        type: "bytes",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "contractAddr",
        type: "string",
      },
      {
        internalType: "bytes",
        name: "key",
        type: "bytes",
      },
    ],
    name: "queryRaw",
    outputs: [
      {
        internalType: "bytes",
        name: "response",
        type: "bytes",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
] as const

export class IWasm__factory {
  static readonly abi = _abi
  static createInterface(): IWasmInterface {
    return new Interface(_abi) as IWasmInterface
  }
  static connect(address: string, runner?: ContractRunner | null): IWasm {
    return new Contract(address, _abi, runner) as unknown as IWasm
  }
}
