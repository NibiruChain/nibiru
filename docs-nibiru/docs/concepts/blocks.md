---
order: 7
canonicalUrl: "https://nibiru.fi/docs/concepts/blocks.html"
---

# Blocks

<!--
2. **Blocks**
   - Questions:
     - [ ] What data does a block contain?
     - [ ] How are blocks linked together in a blockchain?
     - [ ] What is the role of a block header?
     - [ ] How is a block validated?
   - Concepts:
     - [ ] Block structure (header, transactions)
     - [ ] Blockchain immutability
     - [ ] Block validation
     - [ ] Block time
-->

<!--
For advanced section
     - [ ] Related Topics
     - [ ] Merkle Root
     - [ ] Merkle Tree
     - [ ] Merkle Proof
-->

The consensus engine chronicles a sequence of blocks affirmed by a
supermajority of nodes. These blocks form the blockchain, a digital ledger
replicated across all nodes. Blocks, serving as an aggregate of state
transitions, maintain a lineage by referencing their preceding block and
encapsulating transactions. {synopsis}

Blocks are state transtitions. Blocks hold a reference to the previous block
and contain transactions.

Transactions (txs) are state transtitions, each cryptographically signed by
accounts. Txs are made up of transaction messages (TxMsgs).

TxMsgs are *atomic* state transitions. Within a transaction, each TxMsg is
processed entirely or not at all, enuring consistency in state updates.

## Block

A block consists of a header, transactions, votes (the commit),
and a list of evidence of malfeasance (ie. signing conflicting votes).

| Name   | Type | Description | Validation |
| ----   | ---- | ----------- | ---------- |
| Header | [Header](#block-header) | Header corresponding to the block. This field contains information used throughout consensus and other areas of the protocol. To find out what it contains, visit [header] (#block-header) | Must adhere to the validation rules of [header](./#block-header) |
| Data      | [Data](#data)                  | Data contains a list of transactions. The contents of the transactions are unknown to Tendermint.                                                                                      | This field can be empty or populated, but no validation is performed. Applications can perform validation on individual transactions prior to block creation using [checkTx](../abci/abci.md#checktx).
| Evidence   | [EvidenceList](#evidence_list) | Evidence contains a list of infractions committed by validators.                                                                                                                     | Can be empty, but when populated the validations rules from [evidenceList](#evidence_list) apply |
| LastCommit | [Commit](#commit)              | `LastCommit` includes one vote for every validator.  All votes must either be for the previous block, nil or absent. If a vote is for the previous block it must have a valid signature from the corresponding validator. The sum of the voting power of the validators that voted must be greater than 2/3 of the total voting power of the complete validator set. The number of votes in a commit is limited to 10000 (see `types.MaxVotesCount`).                                                                                             | Must be empty for the initial height and must adhere to the validation rules of [commit](#commit).  |

<!-- ## Block Header -->
<!---->
<!-- ## Block Evidence List -->

## BlockID

The `BlockID` contains two distinct Merkle roots of the block. The `BlockID`
includes these two hashes, as well as the number of parts (ie.
`len(MakeParts(block))`)

| Name          | Type                       | Description                                                                                                                     |
|---------------|----------------------------|---------------------------------------------------------------------------------------------------------------------------------|
| Hash          | slice of bytes (`[]byte`)  | MerkleRoot of all the fields in the header (ie. `MerkleRoot(header)`)                                                            |
| PartSetHeader | [PartSetHeader](#partsetheader) | Used for secure gossiping of the block during consensus, is the MerkleRoot of the complete serialized block cut into parts (ie. `MerkleRoot(MakeParts(block))`) |

See [MerkleRoot](./encoding.md#MerkleRoot) for details.

## Block Commit

Commit is a simple wrapper for a list of signatures, with one for each
validator. It also contains the relevant BlockID, height and round:

| Name       | Type                             | Description                                                          | Validation                                                                                               |
|------------|----------------------------------|----------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------|
| Height     | int64                            | Height at which this commit was created.                             | Must be > 0                                                                                              |
| Round      | int32                            | Round that the commit corresponds to.                                | Must be > 0                                                                                              |
| BlockID    | [BlockID](#blockid)              | The blockID of the corresponding block.                              | Must adhere to the validation rules of [BlockID](#blockid).                                              |
| Signatures | Array of [CommitSig](#commitsig) | Array of commit signatures that correspond to current validator set. | Length of signatures must be > 0 and adhere to the validation of each individual [Commitsig](#commitsig) |

## Consensus Vote

A vote is a signed message from a validator for a particular block. The vote
includes information about the validator signing it. When stored in the
blockchain or propagated over the network, votes are encoded in Protobuf.

| Name               | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| Type               | The type of message the vote refers to. Must be `PrevoteType` or `PrecommitType`|
| Height             | Height for which this vote was created for                                  |
| Round              | Round that the commit corresponds to.                                       |
| BlockID            | The blockID of the corresponding block.                                     |
| Timestamp          | Timestamp represents the time at which a validator signed.                  |
| ValidatorAddress   | Address of the validator                                                    |
| ValidatorIndex     | Index at a specific block height corresponding to the Index of the validator in the set.|
| Signature          | Signature by the validator if they participated in consensus for the associated block.|
| Extension          | Vote extension provided by the Application running at the validator's node. |
| ExtensionSignature | Signature for the extension                                                 |
