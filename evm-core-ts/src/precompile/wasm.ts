import type { HexAddr } from "./precompile"

/** Address of Nibiru's Wasm precompiled contract */
export const ADDR_WASM_PRECOMPILE: HexAddr =
  "0x0000000000000000000000000000000000000802"

/** Contract ABI for Nibiru's Wasm precompiled contract.  */
export const ABI_WASM_PRECOMPILE = [
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
