/* Autogenerated file. Do not edit manually. */
/* tslint:disable */
/* eslint-disable */
import type {
  AddressLike,
  BaseContract,
  BigNumberish,
  BytesLike,
  ContractMethod,
  ContractRunner,
  EventFragment,
  FunctionFragment,
  Interface,
  Listener,
  Result,
} from "ethers"

import type {
  TypedContractEvent,
  TypedContractMethod,
  TypedDeferredTopicFilter,
  TypedEventLog,
  TypedListener,
  TypedLogDescription,
} from "./common"

export declare namespace IFunToken {
  export type FunTokenStruct = { erc20: AddressLike; bankDenom: string }

  export type FunTokenStructOutput = [erc20: string, bankDenom: string] & {
    erc20: string
    bankDenom: string
  }

  export type NibiruAccountStruct = {
    ethAddr: AddressLike
    bech32Addr: string
  }

  export type NibiruAccountStructOutput = [
    ethAddr: string,
    bech32Addr: string,
  ] & { ethAddr: string; bech32Addr: string }
}

export interface IFunTokenInterface extends Interface {
  getFunction(
    nameOrSignature:
      | "balance"
      | "bankBalance"
      | "bankMsgSend"
      | "sendToBank"
      | "sendToEvm"
      | "whoAmI",
  ): FunctionFragment

  getEvent(nameOrSignatureOrTopic: "AbciEvent"): EventFragment

  encodeFunctionData(
    functionFragment: "balance",
    values: [AddressLike, AddressLike],
  ): string
  encodeFunctionData(
    functionFragment: "bankBalance",
    values: [AddressLike, string],
  ): string
  encodeFunctionData(
    functionFragment: "bankMsgSend",
    values: [string, string, BigNumberish],
  ): string
  encodeFunctionData(
    functionFragment: "sendToBank",
    values: [AddressLike, BigNumberish, string],
  ): string
  encodeFunctionData(
    functionFragment: "sendToEvm",
    values: [string, BigNumberish, string],
  ): string
  encodeFunctionData(functionFragment: "whoAmI", values: [string]): string

  decodeFunctionResult(functionFragment: "balance", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "bankBalance", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "bankMsgSend", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "sendToBank", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "sendToEvm", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "whoAmI", data: BytesLike): Result
}

export namespace AbciEventEvent {
  export type InputTuple = [eventType: string, abciEvent: string]
  export type OutputTuple = [eventType: string, abciEvent: string]
  export interface OutputObject {
    eventType: string
    abciEvent: string
  }
  export type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>
  export type Filter = TypedDeferredTopicFilter<Event>
  export type Log = TypedEventLog<Event>
  export type LogDescription = TypedLogDescription<Event>
}

export interface IFunToken extends BaseContract {
  connect(runner?: ContractRunner | null): IFunToken
  waitForDeployment(): Promise<this>

  interface: IFunTokenInterface

  queryFilter<TCEvent extends TypedContractEvent>(
    event: TCEvent,
    fromBlockOrBlockhash?: string | number | undefined,
    toBlock?: string | number | undefined,
  ): Promise<Array<TypedEventLog<TCEvent>>>
  queryFilter<TCEvent extends TypedContractEvent>(
    filter: TypedDeferredTopicFilter<TCEvent>,
    fromBlockOrBlockhash?: string | number | undefined,
    toBlock?: string | number | undefined,
  ): Promise<Array<TypedEventLog<TCEvent>>>

  on<TCEvent extends TypedContractEvent>(
    event: TCEvent,
    listener: TypedListener<TCEvent>,
  ): Promise<this>
  on<TCEvent extends TypedContractEvent>(
    filter: TypedDeferredTopicFilter<TCEvent>,
    listener: TypedListener<TCEvent>,
  ): Promise<this>

  once<TCEvent extends TypedContractEvent>(
    event: TCEvent,
    listener: TypedListener<TCEvent>,
  ): Promise<this>
  once<TCEvent extends TypedContractEvent>(
    filter: TypedDeferredTopicFilter<TCEvent>,
    listener: TypedListener<TCEvent>,
  ): Promise<this>

  listeners<TCEvent extends TypedContractEvent>(
    event: TCEvent,
  ): Promise<Array<TypedListener<TCEvent>>>
  listeners(eventName?: string): Promise<Array<Listener>>
  removeAllListeners<TCEvent extends TypedContractEvent>(
    event?: TCEvent,
  ): Promise<this>

  balance: TypedContractMethod<
    [who: AddressLike, funtoken: AddressLike],
    [
      [
        bigint,
        bigint,
        IFunToken.FunTokenStructOutput,
        IFunToken.NibiruAccountStructOutput,
      ] & {
        erc20Balance: bigint
        bankBalance: bigint
        token: IFunToken.FunTokenStructOutput
        whoAddrs: IFunToken.NibiruAccountStructOutput
      },
    ],
    "nonpayable"
  >

  bankBalance: TypedContractMethod<
    [who: AddressLike, bankDenom: string],
    [
      [bigint, IFunToken.NibiruAccountStructOutput] & {
        bankBalance: bigint
        whoAddrs: IFunToken.NibiruAccountStructOutput
      },
    ],
    "nonpayable"
  >

  bankMsgSend: TypedContractMethod<
    [to: string, bankDenom: string, amount: BigNumberish],
    [boolean],
    "nonpayable"
  >

  sendToBank: TypedContractMethod<
    [erc20: AddressLike, amount: BigNumberish, to: string],
    [bigint],
    "nonpayable"
  >

  sendToEvm: TypedContractMethod<
    [bankDenom: string, amount: BigNumberish, to: string],
    [bigint],
    "nonpayable"
  >

  whoAmI: TypedContractMethod<
    [who: string],
    [IFunToken.NibiruAccountStructOutput],
    "nonpayable"
  >

  getFunction<T extends ContractMethod = ContractMethod>(
    key: string | FunctionFragment,
  ): T

  getFunction(nameOrSignature: "balance"): TypedContractMethod<
    [who: AddressLike, funtoken: AddressLike],
    [
      [
        bigint,
        bigint,
        IFunToken.FunTokenStructOutput,
        IFunToken.NibiruAccountStructOutput,
      ] & {
        erc20Balance: bigint
        bankBalance: bigint
        token: IFunToken.FunTokenStructOutput
        whoAddrs: IFunToken.NibiruAccountStructOutput
      },
    ],
    "nonpayable"
  >
  getFunction(nameOrSignature: "bankBalance"): TypedContractMethod<
    [who: AddressLike, bankDenom: string],
    [
      [bigint, IFunToken.NibiruAccountStructOutput] & {
        bankBalance: bigint
        whoAddrs: IFunToken.NibiruAccountStructOutput
      },
    ],
    "nonpayable"
  >
  getFunction(
    nameOrSignature: "bankMsgSend",
  ): TypedContractMethod<
    [to: string, bankDenom: string, amount: BigNumberish],
    [boolean],
    "nonpayable"
  >
  getFunction(
    nameOrSignature: "sendToBank",
  ): TypedContractMethod<
    [erc20: AddressLike, amount: BigNumberish, to: string],
    [bigint],
    "nonpayable"
  >
  getFunction(
    nameOrSignature: "sendToEvm",
  ): TypedContractMethod<
    [bankDenom: string, amount: BigNumberish, to: string],
    [bigint],
    "nonpayable"
  >
  getFunction(
    nameOrSignature: "whoAmI",
  ): TypedContractMethod<
    [who: string],
    [IFunToken.NibiruAccountStructOutput],
    "nonpayable"
  >

  getEvent(
    key: "AbciEvent",
  ): TypedContractEvent<
    AbciEventEvent.InputTuple,
    AbciEventEvent.OutputTuple,
    AbciEventEvent.OutputObject
  >

  filters: {
    "AbciEvent(string,string)": TypedContractEvent<
      AbciEventEvent.InputTuple,
      AbciEventEvent.OutputTuple,
      AbciEventEvent.OutputObject
    >
    AbciEvent: TypedContractEvent<
      AbciEventEvent.InputTuple,
      AbciEventEvent.OutputTuple,
      AbciEventEvent.OutputObject
    >
  }
}
