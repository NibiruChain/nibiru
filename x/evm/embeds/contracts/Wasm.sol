// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

address constant WASM_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000802;

IWasm constant WASM_PRECOMPILE = IWasm(WASM_PRECOMPILE_ADDRESS);

interface IWasm {
  struct BankCoin {
    string denom;
    uint256 amount;
  }

  /// @notice Invoke a contract's "ExecuteMsg", which corresponds to
  /// "wasm/types/MsgExecuteContract". This enables arbitrary smart contract
  /// execution using the Wasm VM from the EVM. 
  /// @param contractAddr nibi-prefixed Bech32 address of the wasm contract
  /// @param msgArgs JSON encoded wasm execute invocation
  /// @param funds Optional funds to supply during the execute call. It's
  /// uncommon to use this field, so you'll pass an empty array most of the time.
  /// @dev The three non-struct arguments are more gas efficient than encoding a
  /// single argument as a WasmExecuteMsg.
  function execute(
    string memory contractAddr,
    bytes memory msgArgs,
    BankCoin[] memory funds
  ) payable external returns (bytes memory response);

  struct WasmExecuteMsg {
    string contractAddr;
    bytes msgArgs;
    BankCoin[] funds;
  }

  /// @notice Identical to "execute", except for multiple contract calls.
  function executeMulti(
    WasmExecuteMsg[] memory executeMsgs
  ) payable external returns (bytes[] memory responses);

  /// @notice Query the public API of another contract at a known address (with
  /// known ABI). 
  /// Implements smart query, the "WasmQuery::Smart" variant from "cosmwas_std".
  /// @param contractAddr nibi-prefixed Bech32 address of the wasm contract
  /// @param req JSON encoded query request
  /// @return response Returns whatever type the contract returns (caller should
  /// know), wrapped in a JSON encoded contract result.
  function query(
    string memory contractAddr, 
    bytes memory req
  ) external view returns (bytes memory response);

  /// @notice Query the raw kv-store of the contract. 
  /// Implements raw query, the "WasmQuery::Raw" variant from "cosmwas_std".
  /// @param contractAddr nibi-prefixed Bech32 address of the wasm contract
  /// @param req JSON encoded query request
  /// @return response The raw, unparsed data stored at that key, which may be an
  /// empty vector if data is not present.
  function queryRaw(
    string memory contractAddr, 
    bytes memory req
  ) external view returns (bytes memory response);

  /// @notice InstantiateContract creates a new smart contract instance for the
  /// given code id.
  function instantiate(
    string memory admin,
    uint64 codeID,
    bytes memory msgArgs,
    string memory label,
    BankCoin[] memory funds
  ) payable external returns (string memory contractAddr, bytes memory data);

}
