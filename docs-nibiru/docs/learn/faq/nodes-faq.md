---
order: 12
# order: false # TODO -->
---

# Nodes Explained

On Nibiru Chain, the term "network" encompasses a collective system of
interconnected nodes. These nodes, which are essentially individual computers or
servers, collaboratively maintain and validate the blockchain's ongoing
operations. {synopsis}

## Q: What is the "Network" in a Blockchain Context?

  The network in a blockchain context is a distributed, peer-to-peer (P2P) ecosystem of nodes that collectively uphold the blockchain's ledger, rules, and consensus mechanism. This decentralized nature ensures no single point of failure or control, enhancing the blockchain's resilience and security.

## Q: How Do Nodes Communicate With Each Other?

  Communication among nodes in a blockchain network, such as Nibiru Chain, is primarily managed through a P2P networking model. In this model, nodes connect directly with each other without a central coordinating server. They share information like transaction data, blocks, and consensus-related messages to maintain the blockchain's state consistently across all nodes.

## Q: What Protocols Are Used for Network Communication?

- **Remote Procedure Call (RPC)**: RPC allows nodes to remotely execute predefined operations. In the context of blockchain, it's often used for querying blockchain data or submitting transactions.
- **Gossip Protocol**: This protocol is instrumental in disseminating information (like transactions and blocks) across the network. It works by having nodes relay information to a few other nodes, which in turn relay it further, rapidly spreading the data network-wide.

### Key Concepts: Blockchain Networking

- **P2P Networking**: This is the core of decentralized communication in
  blockchain, where nodes exchange data directly without a central intermediary.
  It's fundamental for distributing the blockchain's workload and data.
- **Network Protocols (Gossip and RPC)**: These are critical for node
  communication. The Gossip Protocol efficiently spreads information across the
  network, while RPC allows nodes to request specific data or actions from each
  other.
- **Network Security**: Ensures the protection of the blockchain against
  malicious attacks and unauthorized access, maintaining the integrity and
  trustworthiness of the data.
- **Performance Metrics (Latency and Throughput)**: Latency refers to the time
  taken for data to travel across the network, while throughput is the amount of
  data processed in a given timeframe. Both are crucial for evaluating the
  efficiency and capacity of the blockchain network.

## Full Nodes

Full nodes are the backbone of a blockchain network like Nibiru Chain, playing a
pivotal role in its functionality and security.

## Q: What Is a Full Node and What Does It Do?

A full node is a crucial component of the blockchain network that fully validates
transactions and blocks against the blockchain's rules. Unlike lightweight or
partial nodes, full nodes maintain a complete copy of the blockchain ledger,
contributing to the network's robustness and integrity. They participate in the
consensus process by relaying and validating information, though they do not
necessarily engage in creating new blocks unless they are also validator nodes.

## Q: Why Are Full Nodes Important for the Network?

- **Decentralization**: They help to distribute the blockchain's data, reducing
  reliance on any single entity and enhancing network resilience.
- **Security and Integrity**: Full nodes enforce the blockchain's rules, ensuring
  that only valid transactions and blocks are accepted.
- **Network Health**: By maintaining complete copies of the blockchain ledger,
  full nodes contribute to the overall health and accessibility of the
  blockchain's historical data.

## Q: What Are the Requirements to Run a Full Node?

- **Hardware**: Adequate computing power and storage capacity to handle the
  entire blockchain ledger and ongoing transactions.
- **Network Connectivity**: A stable and reliable internet connection to stay in
  sync with the blockchain network.
- **Software**: The relevant blockchain software (in this case, for Nibiru Chain)
  configured to run a full node.

### Key Concepts: Full Nodes

- **Node Types (Validator, Full, Light)**: These refer to different roles
  within the blockchain network. Validators participate in block creation and
  consensus, full nodes validate all data and maintain a complete ledger copy,
  and light nodes offer lightweight, query-focused functionalities.
- **Network Decentralization**: This principle is upheld by full nodes that
  distribute and store the blockchain's data, reducing the reliance on central
  entities and enhancing the resilience of the network.
- **Blockchain Storage**: Full nodes store the entire blockchain ledger. This
  is vital for the redundancy and reliability of network data, ensuring that
  the complete history of the blockchain is preserved and accessible.

## Validator Nodes

**Full Nodes**: These nodes maintain a complete copy of the blockchain ledger and
independently validate each block and transaction against the network's consensus
rules. This validation doesn't mean they participate in the creation or approval
of new blocks (which is the role of validator nodes), but rather, they verify
that the data in received blocks and transactions adheres to the blockchain's
established rules and protocols. If a full node receives a block or transaction
that violates these rules, it will reject and not relay it further. This
contributes to the overall health and security of the network by ensuring that
only correct data is propagated.

**Validator Nodes**: These are a specialized subset of full nodes with additional
responsibilities. Validator nodes not only validate transactions and blocks like
regular full nodes but also participate in the consensus process by proposing new
blocks and voting on block validity. In Proof of Stake (PoS) systems, like many
built with Tendermint, validators are often chosen based on the stake they hold
in the network's native cryptocurrency and are responsible for maintaining the
blockchain's integrity and continuity.

While full nodes are not directly involved in the block creation
process, their role in validating and maintaining a complete and accurate copy of
the blockchain is crucial for the network's reliability and security.
