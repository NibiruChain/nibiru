# nibiru/docs/adr

ADRs help us address and track significant software design choices.

- [Key Terms](#key-terms)
- [Why ADRs Help](#why-adrs-help)
- [Creating ADRs](#creating-adrs)
- [ADR Table of Contents](#adr-table-of-contents)

## Key Terms

| Term | Definition |
| --- | --- |
| Architectural Decision Record (ADR) | Captures a single Architectural Decision (AD), often written informally like personal notes or meeting minutes. The collection of ADRs make up a project's decision log. |
| Architectural Knowledge Management (AKM) | The practice of formally documenting architectural decisions to manage critical project knowledge over time. ADRs are a common AKM technique. | 
| Architectural Decision (AD) | A software design choice that addresses a functional or non-functional requirement. |
| Architecturally Significant Requirement (ASR) | A requirement that has a large, measurable effect on a system's architecture quality and design. |
| Functional Requirement | Describes functionality that a system must provide to fulfill user needs. These define specific behaviors and outcomes that the system should have. For example calculating taxes on a purchase or sending a message. |
| Non-functional Requirement | Describes quality attributes and constraints on a system. Non-functional requirements do not specify behaviors - instead they constrain functional requirements by setting performance metrics, reliability targets, etc. For example - the system should process 95% of transactions in under 1 second. |

## Why ADRs Help

- Docs stay up-to-date: By capturing decisions, not ephemeral state, records have
  lasting relevance even as systems evolve.
- Discoverability: ADRs can be tagged and linked to facilitate discovery when
  working in related code.
- Onboard new developers: ADRs capture context and rationale to quickly ramp up
  new team members.

To maximize these benefits, Nibiru Chain ADRs will:

- Contain a tldr summary section
- Cover background and decision details
- ADRs help transform documentation from a chore to an asset! Diligently recording decisions will improve understanding and pass knowledge to future maintainers.
- Use Markdown formatting for README integration
- Link bidirectionally with related code comments

## Creating ADRs 

1. Start Here: [Nibiru Chain ADR Template](./00-adr-template.md): This provides a starting
  template to capture architectural decisions. 
2. Read a few of the other ADRs to gain a feel for them.

Reference Materials:

- [Request for Comments (RFC) Best Practices](https://datatracker.ietf.org/doc/html/rfc2119)

## ADR Table of Contents

- [ADR-001: Separation of Concerns between MsgServer and Keeper ](./01-adr-msg-server-keeper.md)
