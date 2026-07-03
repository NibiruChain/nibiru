// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @notice Interface defining the AbciEvent for interoperability between
/// Ethereum and the ABCI (Application Blockchain Interface).
interface INibiruEvm {
    struct BankCoin {
        string denom;
        uint256 amount;
    }

    /// @notice Event emitted to in precompiled contracts to relay information
    /// from the ABCI to the EVM logs and indexers. Consumers of this event should
    /// decode the `attrs` parameter based on the `eventType` context.
    ///
    /// @param eventType An identifier type of the event, used for indexing.
    /// Event types indexable with CometBFT indexer are in snake case like
    /// "pending_ethereum_tx" or "message", while protobuf typed events use the
    /// proto message name as their event type (e.g.
    /// "eth.evm.v1.EventEthereumTx").
    /// @param abciEvent JSON object string with the event type and fields of an
    /// ABCI event.
    event AbciEvent(string indexed eventType, string abciEvent);
}
