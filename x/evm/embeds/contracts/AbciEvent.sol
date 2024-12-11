// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

/// @notice Interface defining the AbciEvent for interoperability between
/// Ethereum and the ABCI (Application Blockchain Interface).
interface IAbciEventEmitter {
    /// @notice Event emitted to in precompiled contracts to relay information
    /// from the ABCI to the EVM logs and indexers. Consumers of this event should
    /// decode the `attrs` parameter based on the `eventType` context.
    ///
    /// @param eventType The type of the event, used for categorization and indexing.
    /// @param attrs Arbitrary data payload associated with the event, typically
    /// encoding state changes or metadata.
    event AbciEvent(string indexed eventType, bytes attrs);
}
