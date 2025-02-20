import type { HexAddr } from "./precompile"

/** Address of Nibiru's FunToken precompiled contract */
export const ADDR_FUNTOKEN_PRECOMPILE: HexAddr =
  "0x0000000000000000000000000000000000000800"

/** Contract ABI for Nibiru's FunToken (fungible token) precompiled contract.  */
export const ABI_FUNTOKEN_PRECOMPILE = [
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
        internalType: "address",
        name: "who",
        type: "address",
      },
      {
        internalType: "address",
        name: "funtoken",
        type: "address",
      },
    ],
    name: "balance",
    outputs: [
      {
        internalType: "uint256",
        name: "erc20Balance",
        type: "uint256",
      },
      {
        internalType: "uint256",
        name: "bankBalance",
        type: "uint256",
      },
      {
        components: [
          {
            internalType: "address",
            name: "erc20",
            type: "address",
          },
          {
            internalType: "string",
            name: "bankDenom",
            type: "string",
          },
        ],
        internalType: "struct IFunToken.FunToken",
        name: "token",
        type: "tuple",
      },
      {
        components: [
          {
            internalType: "address",
            name: "ethAddr",
            type: "address",
          },
          {
            internalType: "string",
            name: "bech32Addr",
            type: "string",
          },
        ],
        internalType: "struct IFunToken.NibiruAccount",
        name: "whoAddrs",
        type: "tuple",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "who",
        type: "address",
      },
      {
        internalType: "string",
        name: "bankDenom",
        type: "string",
      },
    ],
    name: "bankBalance",
    outputs: [
      {
        internalType: "uint256",
        name: "bankBalance",
        type: "uint256",
      },
      {
        components: [
          {
            internalType: "address",
            name: "ethAddr",
            type: "address",
          },
          {
            internalType: "string",
            name: "bech32Addr",
            type: "string",
          },
        ],
        internalType: "struct IFunToken.NibiruAccount",
        name: "whoAddrs",
        type: "tuple",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "to",
        type: "string",
      },
      {
        internalType: "string",
        name: "bankDenom",
        type: "string",
      },
      {
        internalType: "uint256",
        name: "amount",
        type: "uint256",
      },
    ],
    name: "bankMsgSend",
    outputs: [
      {
        internalType: "bool",
        name: "success",
        type: "bool",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "erc20",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "amount",
        type: "uint256",
      },
      {
        internalType: "string",
        name: "to",
        type: "string",
      },
    ],
    name: "sendToBank",
    outputs: [
      {
        internalType: "uint256",
        name: "sentAmount",
        type: "uint256",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "bankDenom",
        type: "string",
      },
      {
        internalType: "uint256",
        name: "amount",
        type: "uint256",
      },
      {
        internalType: "string",
        name: "to",
        type: "string",
      },
    ],
    name: "sendToEvm",
    outputs: [
      {
        internalType: "uint256",
        name: "sentAmount",
        type: "uint256",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "string",
        name: "who",
        type: "string",
      },
    ],
    name: "whoAmI",
    outputs: [
      {
        components: [
          {
            internalType: "address",
            name: "ethAddr",
            type: "address",
          },
          {
            internalType: "string",
            name: "bech32Addr",
            type: "string",
          },
        ],
        internalType: "struct IFunToken.NibiruAccount",
        name: "whoAddrs",
        type: "tuple",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
] as const
