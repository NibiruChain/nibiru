---
order: false # TODO
---

# Ethereum JSON-RPC

JSON-RPC is a stateless, lightweight remote procedure call (RPC) protocol. It
primarily defines several data structures and the rules for processing them.
Being transport agnostic, JSON-RPC can be used within the same process, over
sockets, HTTP, or various message-passing environments. It uses JSON (RFC 4627)
as its data format.

JSON-RPC Example: eth_call

The JSON-RPC method eth_call allows you to execute messages against smart
contracts without committing a transaction. Typically, you send a transaction to
a node, which then includes it in the mempool, nodes gossip about it, and
eventually, the transaction is included in a block and executed. However,
eth_call lets you send data to a contract and observe the result without creating
a transaction.

## Lifecycle of an Ethereum Call ("eth_call") 

1. The request to execute eth_call is received and interpreted to identify it as
   a call within the Ethereum namespace.
2. The request is then prepared with the necessary transaction details, such as
   the arguments to be passed, the specific block to execute against, and any
optional parameters that might alter the state for this call.
3. These details are converted into a format that the system can process, known
   as an EthCallRequest. This request is then sent to the EVM module's query
client, which is responsible for handling these types of interactions.
4. The EVM module takes the prepared request, translates it into an internal
   message format, and sets up the Ethereum Virtual Machine (EVM) to process this
message. The message includes all the instructions needed to execute the smart
contract call.
5. The EVM then proceeds to execute the message. Depending on the request, it
   either creates a new smart contract or interacts with an existing one. The
process involves running the necessary computations and applying any state
changes required by the contract.


