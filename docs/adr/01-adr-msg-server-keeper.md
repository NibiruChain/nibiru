# ADR-001: Separation of Concerns between MsgServer and Keeper 

## Introduction

In developing Nibiru Chain, built on Cosmos SDK, we have identified design and
development practices that require optimization. This document proposes
methodologies to differentiate between `MsgServer` and `Keeper` in the code and
to improve our action-based testing framework.

## Changelog

- 2022-06-01: Proposed in [NibiruChain/nibiru
  #524](https://github.com/NibiruChain/nibiru/issues/524) by @testinginprod.
- 2023-12-28: Formal ADR drafted and accepted.

## Context

Merging MsgServer and Keeper goes against separation of responsibilities
principles in Cosmos SDK:

- Obscures distinct functions each component should perform
- Complicates code maintenance due to intertwined responsibilities
- Can lead to security vulnerabilities without clear boundaries

The `MsgServer` should focus on request validation while Keeper handles business logic, like a web app's controller and model respectively.

## Decision

This ADR proposes that we separate MsgServer (validation) from Keeper (business
logic).

For example, Some `MsgServer` methods are restricted to the `x/sudo` group,
showing the need for distinct validation and execution.

The format should be the following:

```go
func NewMsgServerImpl(k Keeper) types.MsgServer { return msgServer{k} }

type msgServer struct {
   k Keeper // NOTE: NO EMBEDDING
}

func NewQueryServerImpl(k Keeper) types.QueryServer { return queryServer{k} }

type queryServer struct {
   k Keeper // NOTE: NO EMBEDDING
}
```

Rules to follow:

- When possible the msg/query server should contain no business logic (not always
  possible due to pagination sometimes)
- Focused only at stateless request validation, and conversion from request to
  arguments required for the keeper function call.
- No embedding because it always ends up with name conflicts.

Keepers:

- Must not have references to request formats, API layer should be totally split
  from business logic layer.

## Benefits

- Simplifies and improves our action-based testing framework:
- Removes need to prepare complex permission schemes
- Reduces boilerplate code in tests when using the keeper as a dependency for
  another module by not requiring explicit "module-name/types" imports.

### Concerns About Security and Access Control

Some might argue that sharing Keeper's methods can lead to security risks, mainly
if there are concerns about unauthorized access. This viewpoint stems from the
belief that the `Keeper` should control access, which might lead to apprehensions
about exposing specific methods.

### Clarifying the Role of the Keeper

However, this perspective needs to be revised in the fundamental role of the
`Keeper`. The primary responsibility of the `Keeper` is to maintain a consistent
state within the application rather than controlling access. Access control and
validation of requests are the responsibilities of the `MsgServer`, which acts as
the first line of defense.

### On Function Privacy

Suppose there's a need to share the Keeper with other modules, and concerns arise
about the safety of exposing specific methods. In that case, the preferred
approach is to keep those sensitive methods private. Implementing access and
permission layers within the `Keeper` goes against the principle of separation of
responsibilities and can lead to a more cohesive and secure system. Instead,
ensuring that only the appropriate methods are exposed and keeping others private
aligns with the philosophy of keeping each component focused on its specific
role.

## Concerns

Exposing Keeper methods could enable unauthorized access. However, access control
is the MsgServer’s responsibility. Keeper maintains state consistency.

Best practice is keeping sensitive methods private, not building permission
schemes in Keeper. This aligns with separation of responsibilities.

## Conclusion

Improves code clarity, maintainability, and security.
