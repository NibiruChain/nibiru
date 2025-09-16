---
order: 9
canonicalUrl: "https://nibiru.fi/docs/concepts/events.html"
---

# Block Events

Block events are responses emitted during the execution of transaction messages.
Events make human-readble information more readily available for clients and
indexers without directly affecting the state of the blockchain. {synopsis}

## Difference between Events and TxMsgs

- **Transaction Messages (TxMsg):** A [TxMsg is an action](./tx-msgs.md)
  that a user sends to the blockchain network to perform some stateful
  operation, such as sending tokens from one account to another. These
  transactions are processed by the blockchain's state machine and can alter
  the state of the blockchain. They are broadcasted to the entire network,
  validated, and once agreed upon by the consensus protocol, they are added to
  a block.

- **Events:** On the other hand, events are a way for the application to signal
  that something significant has occurred as a result of processing a
  transaction. They are a response from the application back to the consensus
  engine. Events do not directly alter the state of the blockchain, but they
  provide a mechanism to notify clients about state changes. For example, an
  application could publish an event when a transfer of tokens occurs, with
  attributes detailing the sender, receiver, and amount.


```go
// Event allows application developers to attach additional information to
// ResponseBeginBlock, ResponseEndBlock, ResponseCheckTx and ResponseDeliverTx.
// Later, transactions may be queried using these events.
type Event struct {
    Type       string
    Attributes []EventAttribute // An event attribute is a key-value pair of strings
}
```

## Consistency of Events for Consensus

Events are not part of the consensus-critical data and are not required to be
consistent across all nodes for consensus. Consensus, as in any BFT-based
consensus protocol, depends on the state transitions that result from
processing transactions, not on the events that are emitted as a result of
those state transitions. 

However, it is generally expected that the same transaction will generate the
same events in all instances of the application.

## How Events are Typically Used

 Events are typically used to signal that a significant action has occurred as
 a result of a transaction. Events provide a means for applications to
 communicate meaningful information about transactions to end-users, external
 services, or other parts of the system in an efficient and flexible manner.
 Events are typically used for:

 - **Indexing and Searching:** Events can be indexed by their type and
   attribute, allowing for efficient searching of historical events. This could
   be used, for example, to quickly find all transfer events involving a
   specific account.

 - **Real-time Updates:** Clients can subscribe to certain types of events and
   receive real-time updates whenever such an event occurs. For instance, a
   wallet app could update a user's balance in real-time whenever a transfer
   event involving the user's account occurs.

 - **Audit Trails:** Events can serve as an audit trail for applications,
   allowing developers or users to see what significant actions have occurred
   in the past. 
