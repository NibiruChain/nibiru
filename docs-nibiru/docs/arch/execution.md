---
order: 1
canonicalUrl: "https://nibiru.fi/docs/arch/execution/"
---

<!-- NOTE: This is page can be removed after the canonical gracefully grabs the index -->

# Execution Engine

The execution engine of Nibiru is the overarching component that implements
business logic and manages the "state" that makes Nibiru a state machine. This is
where transactions are processed and disseminated. {synopsis}

| In this Section | Description |
| --- | --- |
| [Nibiru Adaptive Execution](./execution/adaptive-execution.md) | A hybrid approach to parallel execution in high-contention environments. |
| [Parallel Optimistic Execution](./execution/parallel-optimistic.md) | Nibiru leverages parallel optimistic execution to achieve high transaction throughput and reduced latency by overlapping consensus and execution. |
