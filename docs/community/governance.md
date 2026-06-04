---
description: Nibiru is a decentralized blockchain governed by its community members.
order: 6
---

# Governance

Governance is an on-chain process by which members of the Nibiru Ecosystem can come to consensus and influence changes in the protocol by voting on "**proposals**". Participants can stake NIBI to vote on proposals and the amount of NIBI staked corresponds 1-to-1 with voting power.

## Lifecycle of a Proposal

**Right to submit a proposal:** Any NIBI holder, whether bonded or unbonded, can submit proposals by sending a `TxGovProposal` transaction. Once a proposal is submitted, it is identified by its unique `proposalID`.

The governance process is divided into the following steps:

* **Proposal submission:** Proposal is submitted to the blockchain with a deposit. Once deposit reaches the `MinDeposit`, the proposal is confirmed and the voting period opens.
  * **Claiming deposit:** Users that deposited on proposals can recover their deposits if the proposal is accepted OR if the proposal never enters the voting period.
* **Vote:** In the voting period, bonded NIBI holders (i.e. stakers) can then send `TxGovVote` transactions to vote on the proposal. When a threshold amount of support has been reached
  * Inheritance and penalties: Note that delegators inherit their validator's vote if they don't vote themselves.
* If the proposal involves a software upgrade:
  * **Signal:** Validators start signaling that they are ready to switch to the new version.
  * **Switch:** Once more than 75% of validators have signaled that they are ready to switch, their software automatically flips to the new version.

### ⚖️ — Deposit

To prevent spam, proposals must be submitted with a deposit in the coins defined in the `MinDeposit` param. The voting period will not start until the proposal's deposit equals `MinDeposit`.

When a proposal is submitted, it has to be accompanied by a deposit that must be strictly positive, but can be inferior to `MinDeposit`. The submitter doesn't need to pay for the entire deposit on their own. If a proposal's deposit is inferior to `MinDeposit`, other token holders can increase the proposal's deposit by sending a `Deposit` transaction. The deposit is kept in an escrow in the governance `ModuleAccount` until the proposal is finalized (passed or rejected).

Once the proposal's deposit reaches `MinDeposit`, it enters voting period. If proposal's deposit does not reach `MinDeposit` before `MaxDepositPeriod`, proposal closes and nobody can deposit on it anymore.

#### Deposit refund and burn

When a the a proposal finalized, the coins from the deposit are either refunded or burned, according to the final tally of the proposal:

* If the proposal is approved or if it's rejected but _not_ vetoed, deposits will automatically be refunded to their respective depositor (transferred from the governance `ModuleAccount`).
* When the proposal is vetoed with a supermajority, deposits be burned from the governance `ModuleAccount`.

### ⚖️ — Voting

#### Participants

_Participants_ are users that have the right to vote on proposals. On the Cosmos Hub, participants are bonded NIBI holders. Unbonded NIBI holders and other users do not get the right to participate in governance. However, they can submit and deposit on proposals.

Note that some _participants_ can be forbidden to vote on a proposal under a certain validator if the participant has:

* bonded or unbonded NIBI to said validator after proposal entered voting period.
* become a validator after the proposal enters the voting period.

This does not prevent _participant_ to vote with NIBI bonded to other validators. For example, suppose there are two validators: `valA` and `valB`. If a _participant_ bonds some NIBI to `valA` before a proposal enters voting period and bonds other NIBI to `valB` after the proposal enters its voting period, only the vote under `valB` will be forbidden.

#### Voting period

Once a proposal reaches `MinDeposit`, it immediately enters `Voting period`. We define `Voting period` as the interval between the moment the vote opens and the moment the vote closes. `Voting period` should always be shorter than `Unbonding period` to prevent double voting. The initial value of `Voting period` is 2 weeks.

#### Option set

The option set of a proposal refers to the set of choices a participant can choose from when casting its vote.

The initial option set includes the following options:

* `Yes`
* `No`
* `NoWithVeto`
* `Abstain`

`NoWithVeto` counts as `No` but also adds a `Veto` vote. `Abstain` option allows voters to signal that they do not intend to vote in favor or against the proposal but accept the result of the vote.

#### Weighted Votes

ADR-037 introduces the weighted vote feature which allows a staker to split their votes into several voting options. For example, it could use 70% of its voting power to vote Yes and 30% of its voting power to vote No.

Often times the entity owning that address might not be a single individual. For example, a company might have different stakeholders who want to vote differently, and so it makes sense to allow them to split their voting power. Currently, it is not possible for them to do "passthrough voting" and giving their users voting rights over their tokens. However, with this system, exchanges can poll their users for voting preferences, and then vote on-chain proportionally to the results of the poll.

To represent weighted vote on chain, we use the following [Protobuf](https://developers.google.com/protocol-buffers/docs/overview) message.

```protobuf
// WeightedVoteOption defines a unit of vote for vote split.
message WeightedVoteOption {
  VoteOption option = 1;
  string     weight = 2 [
    (gogoproto.customtype) = "github.com/cosmos/cosmos-sdk/types.Dec",
    (gogoproto.nullable)   = false,
    (gogoproto.moretags)   = "yaml:\"weight\""
  ];
}
```

For a weighted vote to be valid, the `options` field must not contain duplicate vote options, and the sum of weights of all options must be equal to 1.

```protobuf
// Vote defines a vote on a governance proposal.
// A Vote consists of a proposal ID, the voter, and the vote option.
message Vote {
  option (gogoproto.goproto_stringer) = false;
  option (gogoproto.equal)            = false;

  uint64 proposal_id = 1 [(gogoproto.moretags) = "yaml:\"proposal_id\""];
  string voter       = 2;
  reserved 3;
  reserved "option";
  repeated WeightedVoteOption options = 4 [(gogoproto.nullable) = false];
}
```

References - Cosmos-SDK `gov` module protos: [\[WeightedVoteOption\]](https://github.com/cosmos/cosmos-sdk/blob/v0.43.0-alpha1/proto/cosmos/gov/v1beta1/gov.proto#L32-L40) [\[Vote\]](https://github.com/cosmos/cosmos-sdk/blob/v0.43.0-alpha1/proto/cosmos/gov/v1beta1/gov.proto#L126-L137)&#x20;

#### Quorum

Quorum is defined as the minimum percentage of voting power that needs to be casted on a proposal for the result to be valid.

#### Threshold

Threshold is defined as the minimum proportion of `Yes` votes (excluding `Abstain` votes) for the proposal to be accepted.

Initially, the threshold is set at 50% with a possibility to veto if more than 1/3rd of votes (excluding `Abstain` votes) are `NoWithVeto` votes. This means that proposals are accepted if the proportion of `Yes` votes (excluding `Abstain` votes) at the end of the voting period is superior to 50% and if the proportion of `NoWithVeto` votes is inferior to 1/3 (excluding `Abstain` votes).

#### Inheritance

If a delegator does not vote, it will inherit its validator vote.

* If the delegator votes before its validator, it will not inherit from the validator's vote.
* If the delegator votes after its validator, it will override its validator vote with its own. If the proposal is urgent, it is possible that the vote will close before delegators have a chance to react and override their validator's vote. This is not a problem, as proposals require more than 2/3rd of the total voting power to pass before the end of the voting period. If more than 2/3rd of validators collude, they can censor the votes of delegators anyway.

#### Validator’s punishment for non-voting

At present, validators are not punished for failing to vote.

#### Governance address

Later, we may add permissioned keys that could only sign txs from certain modules. For the MVP, the `Governance address` will be the main validator address generated at account creation. This address corresponds to a different PrivKey than the Tendermint PrivKey which is responsible for signing consensus messages. Validators thus do not have to sign governance transactions with the sensitive Tendermint PrivKey.

### ⚖️ — Software Upgrades

If proposals are of type `SoftwareUpgradeProposal`, then nodes need to upgrade their software to the new version that was voted. This process is divided in two steps.

#### Signal

After a `SoftwareUpgradeProposal` is accepted, validators are expected to download and install the new version of the software while continuing to run the previous version. Once a validator has downloaded and installed the upgrade, it will start signaling to the network that it is ready to switch by including the proposal's `proposalID` in its _precommits_.

Note: There is only one signal slot per _precommit_. If several `SoftwareUpgradeProposals` are accepted in a short timeframe, a pipeline will form and they will be implemented one after the other in the order that they were accepted.

#### Switch

Once a block contains more than 2/3rd _precommits_ where a common `SoftwareUpgradeProposal` is signaled, all the nodes (including validator nodes, non-validating full nodes and light-nodes) are expected to switch to the new version of the software.
