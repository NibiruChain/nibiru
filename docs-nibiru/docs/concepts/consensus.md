---
order: 11
canonicalUrl: "https://nibiru.fi/docs/concepts/consensus.html"
title: "Consensus Engine"
---

# Consensus Engine (NibiruBFT)

A consensus engine is a mechanism used to validate and agree on the state of the
distributed ledger and the order of state transitions. In essence, it enables
different nodes within a distributed network to agree on a single source of
truth, despite the presence of faulty nodes. {synopsis}


<!-- 
TODO: 2025-01-31: Update image to be up-to-date 
<img src="../img/chain-arch.png" style="border-radius: 16px;"> 
-->

## [NibiruBFT](../arch/nibiru-bft/README.md)

This page serves as a brief introduction to consensus and only briefly introduces
[NibiruBFT, Nibiru's consensus mechanism](../arch/nibiru-bft/README.md).
NibiruBFT was created by Nibiru's research team as part of the Nibiru Lagrange
Point roadmap. We recommend checking out the full documentation on NibiruBFT if
you like diving deep into the details.

NibiruBFT relies on a set of validator node operators that are responsible for
committing new blocks to the chain. Each validator does so by broadcasting votes
that contain cryptographic signatures corresponding to the validator's private
key. 

Key aspects of NibiruBFT:
- **Instant finality**:
  Once a block is proposed and validated, it is committed to the blockchain and
  immediately considered final. Users can be confident their transactions are
  finalized as soon as a block is created, without the need for multiple
  confirmations or waiting periods (unlike blockchains such as Bitcoin and
  Ethereum). This increases security for users.

- **State machine replication**: 
  NibiruBFT is an extension building from Tendermint, which can can be used to
build arbitrary applications through the Application Blockchain Interface (ABCI).
This means it's not just for cryptocurrencies, but any distributed application.

- **Byzantine Fault Tolerance**: 
  Nibiru employs a Byzantine Fault Tolerance (BFT) consensus protocol,
  specifically the **Practical Byzantine Fault Tolerance (PBFT)** approach. This
  ensures all nodes in the network agree on the order of transactions in the
  blockchain, even when up to one-third of the nodes are Byzantine (i.e.,
  malicious or faulty).  
  PBFT uses a two-phase commit process to achieve consensus, where
  nodes first propose a block and then validate it through a series of voting
  rounds. This process ensures all nodes in the network agree on the order of
  transactions and prevents malicious nodes from compromising the network's
  integrity. 

- **Rotating proposers**: To improve fairness, the right to propose the next
  block is rotated among validators based on their stake and the number of
  times they've already proposed a block.

- NibiruBFT is an evolution on CometBFT, inheriting security guarantees from the
  battle-tested Tendermint Core consensus algorithm. 

<!--
1. **Consensus Engine**
   - Questions:
     - How does a consensus engine work?
     - What makes the Tendermint consensus engine unique?
     - How does the consensus engine handle network partition or node failure?
     - How does the consensus engine prevent double-spending?
   - Concepts:
     - Byzantine Fault Tolerance (BFT)
     - Consensus algorithms (e.g., PoW, PoS)
     - Network partition tolerance
     - Double-spending problem
-->

## Consensus Mechanisms

Consensus mechanisms come in various forms, but they generally follow a similar
process:

1. A transaction is initiated and broadcast to the network.
2. Nodes within the network independently validate the transaction based on
   pre-agreed rules.
3. Once validated, the transaction is included in a block.
4. Nodes in the network then agree (reach consensus) on the correct state of
   this block and its place in the blockchain.

The specific rules and procedures used to reach this agreement vary depending
on the type of consensus mechanism used

## Byzantine Fault Tolerance

In a Byzantine Fault Tolerant system like Nibiru, the network can tolerate
up to one-third of nodes being faulty (either failing or acting maliciously).

If a network partition occurs and neither section has more than two-thirds of
the nodes, the network will halt to prevent any forks or double spends. This is
because Tendermint prioritizes safety (consistency) over liveness
(availability).

If a node failure occurs, as long as there are more than two-thirds of the
remaining nodes functioning properly, the network continues to operate. If a
failed node comes back online, it can sync up with the rest of the network and
resume its operations.

<!--
## Double-spending
## Finality Types
-->


## References

- [Survey on Blockchain Networking: Context, State-of-the-Art, Challenges.](http://eprints.cs.univie.ac.at/6942/1/2-csur21crypto.pdf) Doran et al. 2021.
- [Algorand: Scaling Byzantine Agreements for Cryptocurrencies.](https://eprint.iacr.org/2017/454.pdf) 2017. 
- [O(1) Block propagation.](https://gist.github.com/gavinandresen#file-blockpropagation-md) Andresen, G. 2015. 
- ["What is Tendermint"](https://docs.tendermint.com/v0.34/introduction/what-is-tendermint.html)

