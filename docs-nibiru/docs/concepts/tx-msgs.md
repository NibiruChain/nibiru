---
order: 4
canonicalUrl: "https://nibiru.fi/docs/concepts/tx-msgs.html"
---

# Transaction Messages (TxMsgs)

Transaction messages (TxMsgs) are atomic state transitions. {synopsis}

<!--
4. **Transaction Messages (TxMsg)**
   - Questions:
     - What is a transaction message?
     - How is a transaction message different from a transaction?
     - What kind of actions can a transaction message represent?
   - Concepts:
     - Transaction message structure
     - Transaction execution
     - Transaction types (e.g., token transfer, smart contract interaction)

   - Concepts:
     - TODO State changes
     - TODO State machine


3. **Transactions (Tx)**
   - Questions:
     - What constitutes a transaction?
     - How are transactions validated?
     - What happens if a transaction is invalid?
     - How are transactions added to a block?
   - Concepts:
     - Transaction structure
     - Transaction validation
     - Transaction fees
     - Transaction pool
-->

## What is State?

The "state" refers to the current status and data stored by the blockchain,
such as account balances, smart contract data, etc.

Transactions trigger state changes by altering this stored data. For example, a
transaction may transfer tokens from one account to another, changing the token
balance state.

The core function of a blockchain is to process transactions and transition
between states while maintaining consensus and validity.

## State Machine

The blockchain essentially represents a **finite state machine** - a system
that can be in a finite number of states. The finite state machine refers to
the blockchain's core software that processes transactions and transitions
between states. This provides the foundation for the blockchain's transaction
and consensus logic.

- The current state encompasses all the existing data stored on the blockchain
  such as account balances, contract data, etc.

- Transactions trigger transitions between states. For example, transferring
  tokens changes balances. Given the current state and a transaction, the next
  state is deterministically calculated.

- The state machine ensures transactions cause valid state transitions, where
  "valid" is defined by the application logic. Invalid transactions are
  rejected and do not cause a state change.

- The consensus engine ensures all nodes agree on state changes that get
  committed to the chain. The sequence of states and transitions between them
  form the **blockchain**.

## Blocks, Txs, and TxMsgs

- **Blocks** are state transtitions. Blocks hold a reference to the previous
  block and contain transactions.

- **Transactions (txs)** are state transtitions, each cryptographically signed
  by accounts. Txs are made up of transaction messages (TxMsgs).

- **TxMsgs** are *atomic* state transitions. Within a transaction, each TxMsg
  is processed entirely or not at all, enuring consistency in state updates.

- Implemented by `cosmos-sdk/types.Msg`

## Transactions

Transactions are state transition operations that allow users to interact with
the Nibiru blockchain. Transactions are comprised of metadata and one or more
`TxMsgs` that trigger state changes within modules through their Protobuf
`Msg` services.

When users want to make state changes like transferring tokens or interacting
with smart contracts, they create transactions containing appropriate
`TxMsgs`. Each `TxMsg` must be signed using the private key of the account
initiating the action, before the transaction is broadcasted to the network.

A transaction must then be included in a block, validated, and approved by the
network through the Tendermint consensus process. Once a block containing a
transaction is committed, its state changes are finalized and persisted to the
blockchain.

#### Structure

At a high level, transactions contain:

- `TxMsgs` - State transition operations like token transfers, governed by
  application logic.
- Signatures - Cryptographic signatures of all `TxMsgs` by the initiating
  accounts.
- Fee - Transaction fee paid by the initiator, in gas or tokens.
- Memo - Optional memo for the transaction.
- Timeout Height - Block height after which the transaction is invalid.

#### Lifecycle

The lifecycle of a transaction includes:

1. Creation: The user constructs the transaction by adding `TxMsgs` and
   signing.
2. Propagation: The signed transaction is broadcasted to the network.
3. Pooling: The transaction enters the mempool to await inclusion in a block.
4. Validation: The transaction is validated against application logic and
   signatures verified.
5. Inclusion: The transaction is included in a block by the validator that
   proposed that block.
6. Approval: The network approves the block containing the transaction through
   consensus.
7. Execution: Upon finalized consensus, state transitions in the transaction
   are executed.
8. Confirmation: Transaction is considered immutably confirmed once
   sufficiently buried under newer blocks.

