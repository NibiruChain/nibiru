---
order: 3
---

# Node Daemon

The main endpoint of an SDK application is the **daemon client**, otherwise known as the **full-node** client. The full-node runs the state-machine, starting from a genesis file. It connects to peers running the same client in order to receive and relay transactions, block proposals and signatures. {synopsis}

::: tip
Running a node is different from running a Validator. In order to run a Validator, you must create and sync a node, and then upgrade it by staking.
:::

## What it Means to Start a Full-Node

For the Nibiru Chain application, the `nibid start` command starts a full-node.

The full-node consists of two main parts:

- **State-machine**: The application, defined with the Cosmos-SDK.
- **Consensus Engine**: Engine combining the networking and consensus layers that connects to the application via a socket protocol satisfying the [Application BlockChain Interface (ABCI)](https://docs.tendermint.com/v0.34/introduction/what-is-tendermint.html#abci-overview). The consensus engine is responsible for sharing blocks and transactions between nodes and establishing an immutable order of transactions (the blockchain).

Nibiru uses **Tendermint Core** as its consensus engine, which means that the start command is implemented to boot up a Tendermint node. The Tendermint node can be created with app because the latter satisfies the [`abci.Application` interface](https://github.com/tendermint/tendermint/blob/v0.34.0/abci/types/application.go#L7-L32) (given that `app` extends the Cosmos-SDK [`baseapp`](https://docs.cosmos.network/v0.45/core/baseapp.html)).

## Syncing a Node

As part of the this process, Tendermint makes sure that the height of the application (i.e. number of blocks since genesis) is equal to the height of the Tendermint node. The difference between these two heights should always be negative or null.

If the difference between the block height of the TM node and application is strictly negative, Tendermint will replay blocks until the two block heights are equal (see [`NewNode`](https://github.com/tendermint/tendermint/blob/v0.34.21/node/node.go)).

**Genesis Behavior**: Finally, if the height of the application is 0, the Tendermint node will call InitChain on the application to initialize the state from the genesis file.

Once the Tendermint node is instanciated and in sync with the application, the node can be "started".

Upon starting, the node will bootstrap its RPC and P2P server and start dialing peers. During handshake with its peers, if the node realizes they are ahead, it will query all the blocks sequentially in order to catch up. Then, it will wait for new block proposals and block signatures from validators in order to make progress.
