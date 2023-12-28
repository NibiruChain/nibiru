# ADR1: Refactoring MsgServer and Keeper for Clear Separation of Concerns in Nibiru Chain

## Introduction

In developing Nibiru Chain, built on Cosmos SDK, we have identified design and development practices that require optimization. This document proposes methodologies to differentiate between `MsgServer` and `Keeper` in the code and to improve our action-based testing framework.

## Cosmos SDK: `MsgServer` and `Keeper`

### Issues with Combining `MsgServer` and `Keeper`

Merging `MsgServer` and `Keeper` in the Cosmos SDK context goes against the fundamental principle of separation of responsibilities. This blending results in:

- **Role Confusion:** It obscures the distinct functions that each component should ideally perform. `MsgServer` should focus on request validation, including Permissions, while `Keeper` should handle business logic.
- **Maintenance Challenges:** This conflation complicates code maintenance, as intertwined responsibilities make it difficult to isolate and address issues effectively.
- **Security Implications:** The lack of clear boundaries between validation and execution can lead to security vulnerabilities, as ensuring that each layer only handles its intended tasks becomes challenging.

### Analogy with Web Applications

Compared with a web application, the `MsgServer` would act as a controller to validate requests, while the `Keeper` would be equivalent to business logic or the model.

## Specific Case in Nibiru Chain

Some `MsgServer` methods in the Nibiru Chain are restricted to the `Sudo` group, underscoring the need for a clear separation in the validation and execution of requests.

### Proposal for Functional Separation

We propose that the `MsgServer` solely handles validation while the `Keeper` manages the business logic post-validation.

## Benefits of the Refactor in the Action-Based Testing Framework

With the proposed restructuring of `MsgServer` and `Keeper`, we gain significant benefits in our approach to testing, particularly within our action-based testing framework:

### Simplification of Action-Based Tests

The clear separation of responsibilities between `MsgServer` and `Keeper` allows our tests to focus on creating atomic actions without setting up complex scenarios. This eliminates the need for:

- **Preparing Users and Permissions:** There's no longer a requirement to create a user and add it to the `Sudo` group for each test scenario, greatly simplifying the test setup process.

- **Reducing Boilerplate in Tests:** We minimize additional code required to establish preconditions not directly related to the test's objective.

- **Focus on Business Logic:** Tests can concentrate on assessing pure business logic, undistracted by security and permission configurations.

## Conclusion

Separating the `MsgServer` and `Keeper` in developing and testing the Nibiru Chain will significantly improve the code's clarity, maintenance, and security. These improvements reflect our commitment to efficient and robust development, aligned with the best industry practices.
