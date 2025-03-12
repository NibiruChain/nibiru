/* Autogenerated file. Do not edit manually. */
/* tslint:disable */
/* eslint-disable */
import type {
  BaseContract,
  BigNumberish,
  BytesLike,
  ContractMethod,
  ContractRunner,
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
} from "./common"

export interface NibiruOracleChainLinkLikeInterface extends Interface {
  getFunction(
    nameOrSignature:
      | "_decimals"
      | "decimals"
      | "description"
      | "getRoundData"
      | "latestRoundData"
      | "pair"
      | "version",
  ): FunctionFragment

  encodeFunctionData(functionFragment: "_decimals", values?: undefined): string
  encodeFunctionData(functionFragment: "decimals", values?: undefined): string
  encodeFunctionData(
    functionFragment: "description",
    values?: undefined,
  ): string
  encodeFunctionData(
    functionFragment: "getRoundData",
    values: [BigNumberish],
  ): string
  encodeFunctionData(
    functionFragment: "latestRoundData",
    values?: undefined,
  ): string
  encodeFunctionData(functionFragment: "pair", values?: undefined): string
  encodeFunctionData(functionFragment: "version", values?: undefined): string

  decodeFunctionResult(functionFragment: "_decimals", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "decimals", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "description", data: BytesLike): Result
  decodeFunctionResult(
    functionFragment: "getRoundData",
    data: BytesLike,
  ): Result
  decodeFunctionResult(
    functionFragment: "latestRoundData",
    data: BytesLike,
  ): Result
  decodeFunctionResult(functionFragment: "pair", data: BytesLike): Result
  decodeFunctionResult(functionFragment: "version", data: BytesLike): Result
}

export interface NibiruOracleChainLinkLike extends BaseContract {
  connect(runner?: ContractRunner | null): NibiruOracleChainLinkLike
  waitForDeployment(): Promise<this>

  interface: NibiruOracleChainLinkLikeInterface

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

  _decimals: TypedContractMethod<[], [bigint], "view">

  decimals: TypedContractMethod<[], [bigint], "view">

  description: TypedContractMethod<[], [string], "view">

  getRoundData: TypedContractMethod<
    [arg0: BigNumberish],
    [
      [bigint, bigint, bigint, bigint, bigint] & {
        roundId: bigint
        answer: bigint
        startedAt: bigint
        updatedAt: bigint
        answeredInRound: bigint
      },
    ],
    "view"
  >

  latestRoundData: TypedContractMethod<
    [],
    [
      [bigint, bigint, bigint, bigint, bigint] & {
        roundId: bigint
        answer: bigint
        startedAt: bigint
        updatedAt: bigint
        answeredInRound: bigint
      },
    ],
    "view"
  >

  pair: TypedContractMethod<[], [string], "view">

  version: TypedContractMethod<[], [bigint], "view">

  getFunction<T extends ContractMethod = ContractMethod>(
    key: string | FunctionFragment,
  ): T

  getFunction(
    nameOrSignature: "_decimals",
  ): TypedContractMethod<[], [bigint], "view">
  getFunction(
    nameOrSignature: "decimals",
  ): TypedContractMethod<[], [bigint], "view">
  getFunction(
    nameOrSignature: "description",
  ): TypedContractMethod<[], [string], "view">
  getFunction(nameOrSignature: "getRoundData"): TypedContractMethod<
    [arg0: BigNumberish],
    [
      [bigint, bigint, bigint, bigint, bigint] & {
        roundId: bigint
        answer: bigint
        startedAt: bigint
        updatedAt: bigint
        answeredInRound: bigint
      },
    ],
    "view"
  >
  getFunction(nameOrSignature: "latestRoundData"): TypedContractMethod<
    [],
    [
      [bigint, bigint, bigint, bigint, bigint] & {
        roundId: bigint
        answer: bigint
        startedAt: bigint
        updatedAt: bigint
        answeredInRound: bigint
      },
    ],
    "view"
  >
  getFunction(
    nameOrSignature: "pair",
  ): TypedContractMethod<[], [string], "view">
  getFunction(
    nameOrSignature: "version",
  ): TypedContractMethod<[], [bigint], "view">

  filters: {}
}
